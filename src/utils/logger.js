import fs from 'fs';
import path from 'path';

const LOG_LEVELS = {
  INFO: 'INFO',
  SUCCESS: 'SUCCESS',
  ERROR: 'ERROR',
  WARN: 'WARN',
  DEBUG: 'DEBUG',
};

const LOG_DIR = path.join(process.cwd(), 'logs');

if (!fs.existsSync(LOG_DIR)) {
  fs.mkdirSync(LOG_DIR, { recursive: true });
}

function getTimestamp() {
  return new Date().toISOString();
}

function getDateString() {
  const now = new Date();
  return now.toISOString().split('T')[0];
}

function getLogFilePath(isError = false, appName = null) {
  const dateStr = getDateString();
  let filename;
  
  if (appName) {
    const appSlug = appName.toLowerCase().replace(/\s+/g, '-');
    filename = isError ? `${appSlug}-error-${dateStr}.log` : `${appSlug}-${dateStr}.log`;
  } else {
    filename = isError ? `error-${dateStr}.log` : `app-${dateStr}.log`;
  }
  
  return path.join(LOG_DIR, filename);
}

function writeToFile(logMessage, isError = false, appName = null) {
  try {
    const filePath = getLogFilePath(isError, appName);
    fs.appendFileSync(filePath, logMessage + '\n', 'utf8');
    
    const mainFilePath = getLogFilePath(isError);
    fs.appendFileSync(mainFilePath, logMessage + '\n', 'utf8');
  } catch (error) {
    console.error('Failed to write log to file:', error.message);
  }
}

function cleanOldLogs(daysToKeep = 7) {
  try {
    const files = fs.readdirSync(LOG_DIR);
    const now = Date.now();
    const maxAge = daysToKeep * 24 * 60 * 60 * 1000;

    files.forEach(file => {
      const filePath = path.join(LOG_DIR, file);
      const stats = fs.statSync(filePath);
      const age = now - stats.mtime.getTime();

      if (age > maxAge) {
        fs.unlinkSync(filePath);
        console.log(`Deleted old log file: ${file}`);
      }
    });
  } catch (error) {
    console.error('Failed to clean old logs:', error.message);
  }
}

cleanOldLogs();

function formatLog(level, message, metadata = {}) {
  const timestamp = getTimestamp();
  const metaStr = Object.keys(metadata).length > 0 ? JSON.stringify(metadata) : '';
  return `[${timestamp}] [${level}] ${message} ${metaStr}`;
}

export function logInfo(message, metadata = {}) {
  const logMessage = formatLog(LOG_LEVELS.INFO, message, metadata);
  console.log(logMessage);
  writeToFile(logMessage, false, metadata.appName);
}

export function logSuccess(message, metadata = {}) {
  const logMessage = formatLog(LOG_LEVELS.SUCCESS, `✅ ${message}`, metadata);
  console.log(logMessage);
  writeToFile(logMessage, false, metadata.appName);
}

export function logError(message, error = null, metadata = {}) {
  const errorMeta = error ? {
    ...metadata,
    error: error.message,
    stack: error.stack,
  } : metadata;
  const logMessage = formatLog(LOG_LEVELS.ERROR, `❌ ${message}`, errorMeta);
  console.error(logMessage);
  writeToFile(logMessage, false, metadata.appName);
  writeToFile(logMessage, true, metadata.appName);
}

export function logWarn(message, metadata = {}) {
  const logMessage = formatLog(LOG_LEVELS.WARN, `⚠️ ${message}`, metadata);
  console.warn(logMessage);
  writeToFile(logMessage, false, metadata.appName);
}

export function logDebug(message, metadata = {}) {
  const logMessage = formatLog(LOG_LEVELS.DEBUG, `🔍 ${message}`, metadata);
  console.log(logMessage);
  writeToFile(logMessage, false, metadata.appName);
}

export function logFlow(flow, step, metadata = {}) {
  logInfo(`[${flow}] ${step}`, metadata);
}

export function getLogDirectory() {
  return LOG_DIR;
}
