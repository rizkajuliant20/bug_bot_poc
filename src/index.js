import pkg from '@slack/bolt';
const { App } = pkg;
import { config, validateConfig } from './config.js';
import { handleBugReport } from './handlers/bugHandler.js';
import { startNotionPolling } from './services/notionPolling.js';

validateConfig();

const app = new App({
  token: config.slack.botToken,
  signingSecret: config.slack.signingSecret,
  socketMode: true,
  appToken: config.slack.appToken,
  port: config.port,
});

app.event('reaction_added', async ({ event, client }) => {
  console.log('Reaction detected:', event.reaction);
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
        console.log('Bug reaction detected on message:', message.text);
        
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
      console.error('Error handling reaction:', error);
    }
  }
});

app.event('app_mention', async ({ event, client, say }) => {
  const text = event.text.toLowerCase();
  
  if (text.includes('bug') || text.includes('issue') || text.includes('error') || text.includes('problem')) {
    console.log('Bug mention detected:', event.text);
    await handleBugReport(client, event, say);
  } else {
    await say({
      thread_ts: event.ts,
      text: '👋 Hi! I help create bug tickets in Notion with AI diagnosis. Mention me with a bug report containing keywords like "bug", "issue", "error", or "problem" and I\'ll analyze it and create a ticket for you!',
    });
  }
});

app.command('/bug', async ({ command, ack, client }) => {
  await ack();

  const bugDescription = command.text;
  
  if (!bugDescription) {
    await client.chat.postEphemeral({
      channel: command.channel_id,
      user: command.user_id,
      text: '❌ Please provide a bug description. Usage: `/bug [description]`',
    });
    return;
  }

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
  console.error('Slack app error:', error);
});

(async () => {
  try {
    await app.start();
    console.log('⚡️ Slack Bug Bot is running!');
    console.log('Listening for bug reports...');
    
    // Start polling Notion for manually created bugs (every 2 minutes)
    if (config.slack.bugTrackingChannel) {
      startNotionPolling(app.client, 2);
    }
  } catch (error) {
    console.error('Failed to start app:', error);
    process.exit(1);
  }
})();
