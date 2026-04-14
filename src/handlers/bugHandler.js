import { diagnoseBug, generateBugSummary, summarizeThread } from '../services/aiService.js';
import { createBugTicket, getNotionPageUrl } from '../services/notionService.js';
import { getThreadMessages, getUserInfo, getSlackThreadUrl, sendThreadReply } from '../services/slackService.js';
import { config } from '../config.js';
import { logInfo, logSuccess, logError, logFlow } from '../utils/logger.js';

export async function handleBugReport(client, message, say) {
  const { text, user, channel, ts, thread_ts, team } = message;

  const threadTs = thread_ts || ts;
  
  logFlow('BUG_HANDLER', 'Started processing bug report', { 
    user, 
    channel, 
    ts, 
    threadTs,
    messagePreview: text.substring(0, 100) 
  });
  
  try {
    try {
      await client.reactions.add({
        channel: channel,
        timestamp: ts,
        name: 'eyes',
      });
    } catch (reactionError) {
      if (reactionError.data?.error !== 'already_reacted') {
        throw reactionError;
      }
    }

    const bugDescription = text.replace(/<@[A-Z0-9]+>/g, '').trim();
    
    logFlow('BUG_HANDLER', 'Fetching thread context', { channel, threadTs });
    const threadMessages = await getThreadMessages(client, channel, threadTs);
    const reporterName = await getUserInfo(client, user);
    logInfo('Thread context retrieved', { messageCount: threadMessages.length, reporter: reporterName });

    await sendThreadReply(
      client,
      channel,
      threadTs,
      '🤖 Analyzing bug report with AI...'
    );

    logFlow('BUG_HANDLER', 'Starting AI analysis', { descriptionLength: bugDescription.length });
    const diagnosis = await diagnoseBug(bugDescription, threadMessages);
    logInfo('AI diagnosis completed', { severity: diagnosis.severity, category: diagnosis.category, priority: diagnosis.priority });
    
    const title = await generateBugSummary(bugDescription, diagnosis, threadMessages);
    logInfo('Bug title generated', { title });
    
    const threadSummary = await summarizeThread(threadMessages);
    logInfo('Thread summary generated', { hasSummary: !!threadSummary });

    const slackThreadUrl = getSlackThreadUrl(team, channel, threadTs);
    logInfo('Slack thread URL generated', { url: slackThreadUrl });

    logFlow('BUG_HANDLER', 'Creating Notion ticket');
    const notionPage = await createBugTicket({
      title,
      description: bugDescription,
      diagnosis,
      reporter: reporterName,
      slackThreadUrl,
      threadMessages,
      threadSummary,
    });

    const notionUrl = await getNotionPageUrl(notionPage.id);
    logSuccess('Notion ticket created', { notionUrl, pageId: notionPage.id });

    try {
      await client.reactions.remove({
        channel: channel,
        timestamp: ts,
        name: 'eyes',
      });
    } catch (err) {
      // Ignore if reaction doesn't exist
    }

    try {
      await client.reactions.add({
        channel: channel,
        timestamp: ts,
        name: 'white_check_mark',
      });
    } catch (err) {
      if (err.data?.error !== 'already_reacted') {
        console.error('Error adding success reaction:', err);
      }
    }

    const blocks = [
      {
        type: 'section',
        text: {
          type: 'mrkdwn',
          text: `✅ *Bug ticket created in Notion*`,
        },
      },
      {
        type: 'section',
        fields: [
          {
            type: 'mrkdwn',
            text: `*Title:*\n${title}`,
          },
          {
            type: 'mrkdwn',
            text: `*Severity:*\n${diagnosis.severity.toUpperCase()}`,
          },
          {
            type: 'mrkdwn',
            text: `*Category:*\n${diagnosis.category}`,
          },
          {
            type: 'mrkdwn',
            text: `*Priority:*\n${diagnosis.priority}`,
          },
        ],
      },
      {
        type: 'section',
        text: {
          type: 'mrkdwn',
          text: `*🤖 AI Diagnosis:*\n${diagnosis.rootCause}`,
        },
      },
      {
        type: 'section',
        text: {
          type: 'mrkdwn',
          text: `*💡 Suggested Fix:*\n${diagnosis.suggestedFix}`,
        },
      },
      {
        type: 'actions',
        elements: [
          {
            type: 'button',
            text: {
              type: 'plain_text',
              text: 'View in Notion',
            },
            url: notionUrl,
            style: 'primary',
          },
        ],
      },
    ];

    await sendThreadReply(
      client,
      channel,
      threadTs,
      `Bug ticket created: ${notionUrl}`,
      blocks
    );
    logSuccess('Bug ticket response sent to thread');

    // Send notification to bug tracking channel
    if (config.slack.bugTrackingChannel) {
      logFlow('BUG_HANDLER', 'Sending notification to bug tracking channel', { channel: config.slack.bugTrackingChannel });
      try {
        await client.chat.postMessage({
          channel: config.slack.bugTrackingChannel,
          text: `🐛 New bug ticket created: ${title}`,
          blocks: [
            {
              type: 'header',
              text: {
                type: 'plain_text',
                text: '🐛 New Bug Ticket Created',
                emoji: true,
              },
            },
            {
              type: 'section',
              fields: [
                {
                  type: 'mrkdwn',
                  text: `*Title:*\n${title}`,
                },
                {
                  type: 'mrkdwn',
                  text: `*Reporter:*\n${reporterName}`,
                },
                {
                  type: 'mrkdwn',
                  text: `*Severity:*\n${diagnosis.severity.toUpperCase()}`,
                },
                {
                  type: 'mrkdwn',
                  text: `*Priority:*\n${diagnosis.priority}`,
                },
              ],
            },
            {
              type: 'section',
              text: {
                type: 'mrkdwn',
                text: `*Category:* ${diagnosis.category}\n*Platform:* ${diagnosis.platform ? diagnosis.platform.join(', ') : 'N/A'}`,
              },
            },
            {
              type: 'actions',
              elements: [
                {
                  type: 'button',
                  text: {
                    type: 'plain_text',
                    text: '📝 View in Notion',
                    emoji: true,
                  },
                  url: notionUrl,
                  style: 'primary',
                },
                {
                  type: 'button',
                  text: {
                    type: 'plain_text',
                    text: '💬 View Thread',
                    emoji: true,
                  },
                  url: slackThreadUrl,
                },
              ],
            },
          ],
        });
        logSuccess('Bug notification sent to tracking channel', { channel: config.slack.bugTrackingChannel });
      } catch (error) {
        logError('Failed to send bug notification', error, { channel: config.slack.bugTrackingChannel });
      }
    }

  } catch (error) {
    logError('Bug report handling failed', error, { user, channel, ts });
    
    try {
      await client.reactions.add({
        channel: channel,
        timestamp: ts,
        name: 'x',
      });
    } catch (err) {
      if (err.data?.error !== 'already_reacted') {
        logError('Failed to add error reaction', err);
      }
    }

    await sendThreadReply(
      client,
      channel,
      threadTs,
      `❌ Error creating bug ticket: ${error.message}`
    );
  }
}
