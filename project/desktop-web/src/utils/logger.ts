/* eslint-disable no-console */

const isDev = import.meta.env.DEV;

export const logger = {
    debug: (...args: unknown[]): void => {
        if (isDev) {
            console.debug(...args);
        }
    },
    info: (...args: unknown[]): void => {
        if (isDev) {
            console.info(...args);
        }
    },
    warn: (...args: unknown[]): void => {
        console.warn(...args);
    },
    error: (...args: unknown[]): void => {
        console.error(...args);
    },
};
