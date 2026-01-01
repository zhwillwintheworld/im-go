// WebTransport API TypeScript Definitions
// Based on W3C WebTransport specification: https://www.w3.org/TR/webtransport/
// Note: TypeScript 5.9.3 does not fully support WebTransport in lib.dom.d.ts yet,
// so we provide minimal type definitions here.

interface WebTransportOptions {
    /**
     * Server certificate hashes for self-signed certificates (development only)
     */
    serverCertificateHashes?: WebTransportHash[];
}

interface WebTransportHash {
    algorithm: string;
    value: BufferSource;
}

interface WebTransportBidirectionalStream {
    readonly readable: ReadableStream<Uint8Array>;
    readonly writable: WritableStream<Uint8Array>;
}

interface WebTransportCloseInfo {
    closeCode?: number;
    reason?: string;
}

interface WebTransport {
    readonly ready: Promise<void>;
    readonly closed: Promise<WebTransportCloseInfo>;
    readonly datagrams: {
        readonly readable: ReadableStream<Uint8Array>;
        readonly writable: WritableStream<Uint8Array>;
    };

    close(closeInfo?: WebTransportCloseInfo): void;
    createBidirectionalStream(): Promise<WebTransportBidirectionalStream>;
}

declare const WebTransport: {
    prototype: WebTransport;
    new(url: string, options?: WebTransportOptions): WebTransport;
};
