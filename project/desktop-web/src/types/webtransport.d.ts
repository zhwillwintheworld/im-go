// WebTransport API 类型声明
// 参考: https://www.w3.org/TR/webtransport/

interface WebTransportOptions {
    allowPooling?: boolean;
    congestionControl?: 'default' | 'throughput' | 'low-latency';
    requireUnreliable?: boolean;
    serverCertificateHashes?: WebTransportHash[];
}

interface WebTransportHash {
    algorithm: string;
    value: BufferSource;
}

interface WebTransportDatagramDuplexStream {
    readonly readable: ReadableStream<Uint8Array>;
    readonly writable: WritableStream<Uint8Array>;
    readonly maxDatagramSize: number;
    incomingMaxAge: number | null;
    outgoingMaxAge: number | null;
    incomingHighWaterMark: number;
    outgoingHighWaterMark: number;
}

interface WebTransportBidirectionalStream {
    readonly readable: ReadableStream<Uint8Array>;
    readonly writable: WritableStream<Uint8Array>;
}

interface WebTransportSendStreamStats {
    timestamp: DOMHighResTimeStamp;
    bytesWritten: number;
    bytesSent: number;
    bytesAcknowledged: number;
}

interface WebTransportReceiveStreamStats {
    timestamp: DOMHighResTimeStamp;
    bytesReceived: number;
    bytesRead: number;
}

interface WebTransportStats {
    timestamp: DOMHighResTimeStamp;
    bytesSent: number;
    packetsSent: number;
    packetsLost: number;
    numOutgoingStreamsCreated: number;
    numIncomingStreamsCreated: number;
    bytesReceived: number;
    packetsReceived: number;
    smoothedRtt: DOMHighResTimeStamp;
    rttVariation: DOMHighResTimeStamp;
    minRtt: DOMHighResTimeStamp;
    datagamsDropped: number;
    datagramsSent: number;
    datagramsReceived: number;
}

interface WebTransportCloseInfo {
    closeCode?: number;
    reason?: string;
}

interface WebTransport {
    readonly ready: Promise<void>;
    readonly closed: Promise<WebTransportCloseInfo>;
    readonly draining: Promise<void>;
    readonly datagrams: WebTransportDatagramDuplexStream;
    readonly incomingBidirectionalStreams: ReadableStream<WebTransportBidirectionalStream>;
    readonly incomingUnidirectionalStreams: ReadableStream<ReadableStream<Uint8Array>>;
    readonly congestionControl: 'default' | 'throughput' | 'low-latency';
    readonly anticipatedConcurrentIncomingUnidirectionalStreams: number | null;
    readonly anticipatedConcurrentIncomingBidirectionalStreams: number | null;

    close(closeInfo?: WebTransportCloseInfo): void;
    createBidirectionalStream(): Promise<WebTransportBidirectionalStream>;
    createUnidirectionalStream(): Promise<WritableStream<Uint8Array>>;
    getStats(): Promise<WebTransportStats>;
}

declare var WebTransport: {
    prototype: WebTransport;
    new(url: string, options?: WebTransportOptions): WebTransport;
};
