import pino from 'pino';
import { config } from './config';

/**
 * Pino Logger Configuration
 * Optimized for GCP Cloud Run (JSON) and Local Development (Pretty)
 */
export const logger = pino({
  level: config.NODE_ENV === 'test' ? 'silent' : 'info',
  // Map Pino levels to GCP Severity levels
  formatters: {
    level: (label) => {
      return { severity: label.toUpperCase() };
    },
  },
  // GCP Cloud Logging uses 'message' instead of 'msg'
  messageKey: 'message',
  // Use ISO time for GCP
  timestamp: pino.stdTimeFunctions.isoTime,
  transport: config.NODE_ENV === 'development'
    ? {
        target: 'pino-pretty',
        options: {
          colorize: true,
          translateTime: 'SYS:standard',
        },
      }
    : undefined,
});
