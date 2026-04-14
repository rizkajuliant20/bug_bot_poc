const LOG_LEVELS = {
  INFO: 'INFO',
  SUCCESS: 'SUCCESS',
  ERROR: 'ERROR',
  WARN: 'WARN',
  DEBUG: 'DEBUG',
};

function getTimestamp() {
  return new Date().toISOString();
}

function formatLog(level, message, metadata = {}) {
  const timestamp = getTimestamp();
  const metaStr = Object.keys(metadata).length > 0 ? JSON.stringify(metadata) : '';
  return `[${timestamp}] [${level}] ${message} ${metaStr}`;
}

export function logInfo(message, metadata = {}) {
  console.log(formatLog(LOG_LEVELS.INFO, message, metadata));
}

export function logSuccess(message, metadata = {}) {
  console.log(formatLog(LOG_LEVELS.SUCCESS, `✅ ${message}`, metadata));
}

export function logError(message, error = null, metadata = {}) {
  const errorMeta = error ? {
    ...metadata,
    error: error.message,
    stack: error.stack,
  } : metadata;
  console.error(formatLog(LOG_LEVELS.ERROR, `❌ ${message}`, errorMeta));
}

export function logWarn(message, metadata = {}) {
  console.warn(formatLog(LOG_LEVELS.WARN, `⚠️ ${message}`, metadata));
}

export function logDebug(message, metadata = {}) {
  console.log(formatLog(LOG_LEVELS.DEBUG, `🔍 ${message}`, metadata));
}

export function logFlow(flow, step, metadata = {}) {
  logInfo(`[${flow}] ${step}`, metadata);
}
