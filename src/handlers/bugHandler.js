import { diagnoseBug, generateBugSummary, summarizeThread } from '../services/aiService.js';
import { createBugTicket, getNotionPageUrl } from '../services/notionService.js';
import { getThreadMessages, getUserInfo, getSlackThreadUrl, sendThreadReply } from '../services/slackService.js';
import { config } from '../config.js';

export async function handleBugReport(client, message, say) {
  const { text, user, channel, ts, thread_ts, team } = message;

  const threadTs = thread_ts || ts;
  
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
    
    const threadMessages = await getThreadMessages(client, channel, threadTs);
    const reporterName = await getUserInfo(client, user);

    await sendThreadReply(
      client,
      channel,
      threadTs,
      '🤖 Analyzing bug report with AI...'
    );

    const diagnosis = await diagnoseBug(bugDescription, threadMessages);
    const title = await generateBugSummary(bugDescription, diagnosis, threadMessages);
    const threadSummary = await summarizeThread(threadMessages);

    const slackThreadUrl = getSlackThreadUrl(team, channel, threadTs);

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

    // Send notification to bug tracking channel
    if (config.slack.bugTrackingChannel) {
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
        console.log('Bug notification sent to tracking channel');
      } catch (error) {
        console.error('Error sending bug notification:', error);
      }
    }

  } catch (error) {
    console.error('Error handling bug report:', error);
    
    try {
      await client.reactions.add({
        channel: channel,
        timestamp: ts,
        name: 'x',
      });
    } catch (err) {
      if (err.data?.error !== 'already_reacted') {
        console.error('Error adding error reaction:', err);
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
