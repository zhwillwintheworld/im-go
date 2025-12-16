package server

import (
	"context"
	"crypto/tls"
	"log/slog"
	"net/http"
	"sync"

	"github.com/quic-go/quic-go"
	"github.com/quic-go/quic-go/http3"
	"github.com/quic-go/webtransport-go"
	"sudooom.im.access/internal/config"
	"sudooom.im.access/internal/connection"
	"sudooom.im.access/internal/nats"
	"sudooom.im.access/internal/protocol"
	"sudooom.im.access/internal/redis"
	sharedNats "sudooom.im.shared/nats"
)

type Server struct {
	cfg              *config.Config
	natsClient       *nats.Client
	redisClient      *redis.Client
	logger           *slog.Logger
	connMgr          *connection.Manager
	handler          *protocol.Handler
	wtServer         *webtransport.Server
	heartbeatChecker *connection.HeartbeatChecker
	wg               sync.WaitGroup
}

func New(cfg *config.Config, natsClient *nats.Client, redisClient *redis.Client, logger *slog.Logger) *Server {
	connMgr := connection.NewManager()
	handler := protocol.NewHandler(connMgr, natsClient, redisClient, cfg.Server.NodeID, logger)

	return &Server{
		cfg:         cfg,
		natsClient:  natsClient,
		redisClient: redisClient,
		logger:      logger,
		connMgr:     connMgr,
		handler:     handler,
	}
}

func (s *Server) Start(ctx context.Context) error {
	tlsConfig, err := s.loadTLSConfig()
	if err != nil {
		return err
	}

	quicConfig := &quic.Config{
		MaxIdleTimeout:        s.cfg.QUIC.MaxIdleTimeout,
		KeepAlivePeriod:       s.cfg.QUIC.KeepAlivePeriod,
		MaxIncomingStreams:    s.cfg.QUIC.MaxIncomingStreams,
		MaxIncomingUniStreams: s.cfg.QUIC.MaxIncomingUniStreams,
		Allow0RTT:             s.cfg.QUIC.Allow0RTT,
	}

	// 创建 WebTransport 服务器
	s.wtServer = &webtransport.Server{
		H3: http3.Server{
			Addr:       s.cfg.Server.Addr,
			TLSConfig:  tlsConfig,
			QUICConfig: quicConfig,
		},
		CheckOrigin: func(r *http.Request) bool {
			// TODO: 生产环境应该检查 Origin
			return true
		},
	}

	// 设置 HTTP 路由
	mux := http.NewServeMux()
	mux.HandleFunc("/webtransport", func(w http.ResponseWriter, r *http.Request) {
		session, err := s.wtServer.Upgrade(w, r)
		if err != nil {
			s.logger.Error("WebTransport upgrade failed", "error", err)
			return
		}
		s.wg.Add(1)
		go s.handleSession(ctx, session)
	})

	s.wtServer.H3.Handler = mux

	// 订阅 NATS 下行消息
	s.subscribeDownstream()

	// 启动心跳检测器
	s.heartbeatChecker = connection.NewHeartbeatChecker(
		s.connMgr,
		s.cfg.Server.HeartbeatTimeout,
		s.cfg.Server.HeartbeatCheckInterval,
		s.logger,
		func(conn *connection.Connection) {
			// 超时回调：清理用户位置并通知 Logic
			if conn.UserID() > 0 {
				s.redisClient.UnregisterUserLocation(ctx, conn.UserID(), conn.ID())
				s.handler.SendUserOfflineToLogic(conn)
			}
		},
	)
	go s.heartbeatChecker.Start(ctx)

	s.logger.Info("WebTransport server starting", "addr", s.cfg.Server.Addr)

	// 启动服务器
	return s.wtServer.ListenAndServe()
}

func (s *Server) handleSession(ctx context.Context, session *webtransport.Session) {
	defer s.wg.Done()

	c := connection.NewFromWebTransport(session, s.logger)
	s.connMgr.Add(c)
	defer func() {
		// 连接关闭时清理用户位置
		if c.UserID() > 0 {
			s.redisClient.UnregisterUserLocation(ctx, c.UserID(), c.ID())
			s.handler.SendUserOfflineToLogic(c)
		}
		s.connMgr.Remove(c.ID())
	}()

	s.logger.Info("New WebTransport session", "conn_id", c.ID())

	// 处理传入的双向流
	for {
		stream, err := session.AcceptStream(ctx)
		if err != nil {
			s.logger.Debug("Session closed", "conn_id", c.ID())
			return
		}
		go s.handler.HandleStream(ctx, c, stream)
	}
}

func (s *Server) subscribeDownstream() {
	nodeID := s.getNodeID()
	subject := sharedNats.BuildAccessDownstreamSubject(nodeID)

	s.natsClient.Subscribe(subject, func(data []byte) {
		s.handler.HandleDownstream(data)
	})

	// 订阅广播
	s.natsClient.Subscribe(sharedNats.SubjectAccessBroadcast, func(data []byte) {
		s.handler.HandleDownstream(data)
	})

	s.logger.Info("Subscribed to downstream", "subject", subject)
}

func (s *Server) getNodeID() string {
	if s.cfg.Server.NodeID != "" {
		return s.cfg.Server.NodeID
	}
	return "access-1"
}

func (s *Server) loadTLSConfig() (*tls.Config, error) {
	if s.cfg.QUIC.CertFile != "" && s.cfg.QUIC.KeyFile != "" {
		cert, err := tls.LoadX509KeyPair(s.cfg.QUIC.CertFile, s.cfg.QUIC.KeyFile)
		if err != nil {
			return nil, err
		}
		return &tls.Config{
			Certificates: []tls.Certificate{cert},
			NextProtos:   []string{"h3", "webtransport"},
			MinVersion:   tls.VersionTLS13,
		}, nil
	}

	// 开发环境：生成自签名证书
	s.logger.Warn("No TLS certificate configured, using self-signed certificate")
	return generateSelfSignedTLSConfig()
}

// ConnManager 返回连接管理器
func (s *Server) ConnManager() *connection.Manager {
	return s.connMgr
}

func (s *Server) Shutdown() {
	if s.wtServer != nil {
		s.wtServer.Close()
	}
	s.wg.Wait()
}
