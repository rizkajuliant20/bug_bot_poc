import pkg from '@slack/bolt';
const { App } = pkg;
import { config, validateConfig } from './config.js';
import { handleBugReport } from './handlers/bugHandler.js';
import { startNotionPolling } from './services/notionPolling.js';
import { logInfo, logSuccess, logError, logFlow } from './utils/logger.js';

validateConfig();

const app = new App({
  token: config.slack.botToken,
  signingSecret: config.slack.signingSecret,
  socketMode: true,
  appToken: config.slack.appToken,
  port: config.port,
});

app.event('reaction_added', async ({ event, client }) => {
  logFlow('SLACK_EVENT', 'Reaction detected', { reaction: event.reaction, channel: event.item.channel });
  if (event.reaction === 'lady_beetle' || event.reaction === 'ladybug' || event.reaction === 'bug' || event.reaction === 'beetle') {
    try {
      const result = await client.conversations.history({
        channel: event.item.channel,
        latest: event.item.ts,
        limit: 1,
        inclusive: true,
      });

      if (result.messages && result.messages.length > 0) {
        const message = result.messages[0];
        logSuccess('Bug reaction detected - triggering bug handler', { 
          reaction: event.reaction, 
          channel: event.item.channel,
          messagePreview: message.text?.substring(0, 50)
        });
        
        await handleBugReport(client, {
          ...message,
          channel: event.item.channel,
          team: event.item.team || event.team,
        }, async (msg) => {
          await client.chat.postMessage({
            channel: event.item.channel,
            thread_ts: message.ts,
            ...msg,
          });
        });
      }
    } catch (error) {
      logError('Failed to handle reaction event', error, { reaction: event.reaction });
    }
  }
});

app.event('app_mention', async ({ event, client, say }) => {
  logFlow('SLACK_EVENT', 'App mention detected', { user: event.user, channel: event.channel });
  
  const text = event.text.toLowerCase();
  
  if (text.includes('bug') || text.includes('issue') || text.includes('error') || text.includes('problem')) {
    logSuccess('Bug mention detected - triggering bug handler', { 
      user: event.user, 
      channel: event.channel,
      messagePreview: event.text.substring(0, 50)
    });
    await handleBugReport(client, event, say);
  } else {
    logInfo('App mention without bug keywords - sending help message', { user: event.user });
    await say({
      thread_ts: event.ts,
      text: '👋 Hi! I help create bug tickets in Notion with AI diagnosis. Mention me with a bug report containing keywords like "bug", "issue", "error", or "problem" and I\'ll analyze it and create a ticket for you!',
    });
  }
});

app.command('/bug', async ({ command, ack, client }) => {
  await ack();
  
  logFlow('SLACK_EVENT', 'Slash command /bug received', { 
    user: command.user_id, 
    channel: command.channel_id 
  });

  const bugDescription = command.text;
  
  if (!bugDescription) {
    logInfo('Slash command /bug called without description', { user: command.user_id });
    await client.chat.postEphemeral({
      channel: command.channel_id,
      user: command.user_id,
      text: '❌ Please provide a bug description. Usage: `/bug [description]`',
    });
    return;
  }
  
  logSuccess('Slash command /bug processing', { user: command.user_id, descriptionLength: bugDescription.length });

  const message = {
    text: bugDescription,
    user: command.user_id,
    channel: command.channel_id,
    ts: Date.now() / 1000,
    team: command.team_id,
  };

  await handleBugReport(client, message, async (msg) => {
    await client.chat.postMessage({
      channel: command.channel_id,
      ...msg,
    });
  });
});

app.error(async (error) => {
  logError('Slack app error', error);
});

(async () => {
  try {
    await app.start();
    logSuccess('⚡️ Slack Bug Bot is running!');
    logInfo('Listening for bug reports...');
    
    // Start polling Notion for manually created bugs (every 2 minutes)
    if (config.slack.bugTrackingChannel) {
      logInfo('Bug tracking channel configured', { channel: config.slack.bugTrackingChannel });
      startNotionPolling(app.client, 2);
    } else {
      logInfo('Bug tracking channel not configured - polling disabled');
    }
  } catch (error) {
    logError('Failed to start app', error);
    process.exit(1);
  }
})();
