export enum MsgType {
    Heartbeat = 0,
    Auth = 1,
    AuthAck = 2,
    Message = 10,
    MessageAck = 11,
}

export interface Packet {
    msgType: MsgType;
    body: any;
}

export class IMProtocol {
    private static HEADER_SIZE = 6; // 4 bytes length + 2 bytes type

    static encode(msgType: MsgType, data: any): Uint8Array {
        // 1. Serialize body
        let bodyBytes: Uint8Array;
        if (data instanceof Uint8Array) {
            bodyBytes = data;
        } else {
            const jsonStr = JSON.stringify(data);
            const encoder = new TextEncoder();
            bodyBytes = encoder.encode(jsonStr);
        }

        // 2. Create buffer
        const buffer = new ArrayBuffer(this.HEADER_SIZE + bodyBytes.length);
        const view = new DataView(buffer);

        // 3. Write Header
        // Length (4 bytes, Big Endian)
        view.setUint32(0, bodyBytes.length, false);
        // MsgType (2 bytes, Big Endian)
        view.setUint16(4, msgType, false);

        // 4. Write Body
        const bytes = new Uint8Array(buffer);
        bytes.set(bodyBytes, this.HEADER_SIZE);

        return bytes;
    }

    static decode(buffer: ArrayBuffer): Packet | null {
        if (buffer.byteLength < this.HEADER_SIZE) {
            return null;
        }

        const view = new DataView(buffer);
        const length = view.getUint32(0, false);
        const msgType = view.getUint16(4, false);

        if (buffer.byteLength < this.HEADER_SIZE + length) {
            console.warn('Incomplete packet');
            return null;
        }

        // Extract body
        const bodyBytes = new Uint8Array(buffer, this.HEADER_SIZE, length);
        const decoder = new TextDecoder();
        const jsonStr = decoder.decode(bodyBytes);

        try {
            const body = JSON.parse(jsonStr);
            return { msgType, body };
        } catch (e) {
            console.error('Failed to parse message body', e);
            return { msgType, body: null };
        }
    }
}
