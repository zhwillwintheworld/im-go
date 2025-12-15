package server

import (
	"context"
	"crypto/tls"
	"sync"

	"github.com/example/im-access/internal/config"
	"github.com/example/im-access/internal/connection"
	"github.com/example/im-access/internal/nats"
	"github.com/example/im-access/internal/protocol"
	"github.com/quic-go/quic-go"
	"go.uber.org/zap"
)

type Server struct {
	cfg        *config.Config
	natsClient *nats.Client
	logger     *zap.Logger
	connMgr    *connection.Manager
	handler    *protocol.Handler
	listener   *quic.Listener
	wg         sync.WaitGroup
}

func New(cfg *config.Config, natsClient *nats.Client, logger *zap.Logger) *Server {
	connMgr := connection.NewManager()
	handler := protocol.NewHandler(connMgr, natsClient, logger)

	return &Server{
		cfg:        cfg,
		natsClient: natsClient,
		logger:     logger,
		connMgr:    connMgr,
		handler:    handler,
	}
}

func (s *Server) Start(ctx context.Context) error {
	tlsConfig := s.generateTLSConfig()

	quicConfig := &quic.Config{
		MaxIdleTimeout:        s.cfg.QUIC.MaxIdleTimeout,
		KeepAlivePeriod:       s.cfg.QUIC.KeepAlivePeriod,
		MaxIncomingStreams:    s.cfg.QUIC.MaxIncomingStreams,
		MaxIncomingUniStreams: s.cfg.QUIC.MaxIncomingUniStreams,
		Allow0RTT:             s.cfg.QUIC.Allow0RTT,
	}

	listener, err := quic.ListenAddr(s.cfg.Server.Addr, tlsConfig, quicConfig)
	if err != nil {
		return err
	}
	s.listener = listener

	// 订阅 NATS 下行消息
	s.subscribeDownstream()

	// 接受连接
	for {
		select {
		case <-ctx.Done():
			return nil
		default:
			conn, err := listener.Accept(ctx)
			if err != nil {
				s.logger.Error("Failed to accept connection", zap.Error(err))
				continue
			}
			s.wg.Add(1)
			go s.handleConnection(ctx, conn)
		}
	}
}

func (s *Server) handleConnection(ctx context.Context, conn quic.Connection) {
	defer s.wg.Done()

	c := connection.New(conn, s.logger)
	s.connMgr.Add(c)
	defer s.connMgr.Remove(c.ID())

	s.logger.Info("New connection", zap.Int64("conn_id", c.ID()))

	for {
		stream, err := conn.AcceptStream(ctx)
		if err != nil {
			s.logger.Debug("Connection closed", zap.Int64("conn_id", c.ID()))
			return
		}
		go s.handler.HandleStream(ctx, c, stream)
	}
}

func (s *Server) subscribeDownstream() {
	nodeID := s.getNodeID()
	subject := "im.access." + nodeID + ".downstream"

	s.natsClient.Subscribe(subject, func(data []byte) {
		s.handler.HandleDownstream(data)
	})

	s.logger.Info("Subscribed to downstream", zap.String("subject", subject))
}

func (s *Server) getNodeID() string {
	// TODO: 从配置或环境变量获取
	return "access-1"
}

func (s *Server) generateTLSConfig() *tls.Config {
	// TODO: 从配置加载证书
	return &tls.Config{
		NextProtos: []string{"im-access"},
		MinVersion: tls.VersionTLS13,
	}
}

func (s *Server) Shutdown() {
	if s.listener != nil {
		s.listener.Close()
	}
	s.wg.Wait()
}
