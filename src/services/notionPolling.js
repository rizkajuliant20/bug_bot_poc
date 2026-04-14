import { Client } from '@notionhq/client';
import { config } from '../config.js';
import fs from 'fs';
import path from 'path';

const notion = new Client({
  auth: config.notion.apiKey,
});

const TRACKING_FILE = path.join(process.cwd(), '.notion-tracking.json');

// Load tracked bug IDs
function loadTrackedBugs() {
  try {
    if (fs.existsSync(TRACKING_FILE)) {
      const data = fs.readFileSync(TRACKING_FILE, 'utf8');
      return new Set(JSON.parse(data));
    }
  } catch (error) {
    console.error('Error loading tracked bugs:', error);
  }
  return new Set();
}

// Save tracked bug IDs
function saveTrackedBugs(trackedBugs) {
  try {
    fs.writeFileSync(TRACKING_FILE, JSON.stringify([...trackedBugs]), 'utf8');
  } catch (error) {
    console.error('Error saving tracked bugs:', error);
  }
}

// Poll Notion database for new bugs
export async function pollNotionForNewBugs(slackClient) {
  const trackedBugs = loadTrackedBugs();
  
  try {
    // Query Notion database for recent bugs (last 10 minutes)
    const tenMinutesAgo = new Date(Date.now() - 10 * 60 * 1000).toISOString();
    
    const response = await notion.databases.query({
      database_id: config.notion.databaseId,
      filter: {
        timestamp: 'created_time',
        created_time: {
          after: tenMinutesAgo,
        },
      },
      sorts: [
        {
          timestamp: 'created_time',
          direction: 'descending',
        },
      ],
    });

    const newBugs = [];
    
    for (const page of response.results) {
      // Skip if already tracked
      if (trackedBugs.has(page.id)) {
        continue;
      }

      // Extract bug info
      const title = page.properties.Title?.title?.[0]?.plain_text || 'Untitled Bug';
      const tags = page.properties.Tags?.multi_select?.map(t => t.name.toLowerCase()) || [];
      const severity = page.properties.Severity?.select?.name || 'Unknown';
      const priority = page.properties.Priority?.select?.name || 'Unknown';
      const category = page.properties.Category?.select?.name || 'Unknown';
      const reporter = page.properties.Reporter?.rich_text?.[0]?.plain_text || 'Unknown';
      const platform = page.properties.Platform?.multi_select?.map(p => p.name) || [];
      const slackThread = page.properties['Slack Thread']?.url || null;

      // Skip if it doesn't have 'bug' tag
      if (!tags.includes('bug')) {
        trackedBugs.add(page.id); // Still track it to avoid checking again
        continue;
      }

      // Skip if it has Slack Thread URL (automation-created)
      if (slackThread) {
        trackedBugs.add(page.id); // Track it but don't notify
        continue;
      }

      newBugs.push({
        id: page.id,
        title,
        severity,
        priority,
        category,
        reporter,
        platform,
        slackThread,
        notionUrl: `https://notion.so/${page.id.replace(/-/g, '')}`,
        createdTime: page.created_time,
      });

      // Mark as tracked
      trackedBugs.add(page.id);
    }

    // Save updated tracking
    if (newBugs.length > 0) {
      saveTrackedBugs(trackedBugs);
      
      // Send notifications for all new bugs with 'bug' tag
      for (const bug of newBugs) {
        await sendBugNotification(slackClient, bug);
      }
    }

    return newBugs;
  } catch (error) {
    console.error('Error polling Notion:', error);
    return [];
  }
}

// Send Slack notification for manually created bug
async function sendBugNotification(slackClient, bug) {
  if (!config.slack.bugTrackingChannel) {
    return;
  }

  try {
    await slackClient.chat.postMessage({
      channel: config.slack.bugTrackingChannel,
      text: `🐛 New bug ticket created manually: ${bug.title}`,
      blocks: [
        {
          type: 'header',
          text: {
            type: 'plain_text',
            text: '🐛 New Bug Ticket',
            emoji: true,
          },
        },
        {
          type: 'section',
          fields: [
            {
              type: 'mrkdwn',
              text: `*Title:*\n${bug.title}`,
            },
            {
              type: 'mrkdwn',
              text: `*Reporter:*\n${bug.reporter}`,
            },
            {
              type: 'mrkdwn',
              text: `*Severity:*\n${bug.severity}`,
            },
            {
              type: 'mrkdwn',
              text: `*Priority:*\n${bug.priority}`,
            },
          ],
        },
        {
          type: 'section',
          text: {
            type: 'mrkdwn',
            text: `*Category:* ${bug.category}\n*Platform:* ${bug.platform.length > 0 ? bug.platform.join(', ') : 'N/A'}`,
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
              url: bug.notionUrl,
              style: 'primary',
            },
          ],
        },
      ],
    });
    
    console.log(`Notification sent for manually created bug: ${bug.title}`);
  } catch (error) {
    console.error('Error sending bug notification:', error);
  }
}

// Start polling interval
export function startNotionPolling(slackClient, intervalMinutes = 2) {
  console.log(`Starting Notion polling every ${intervalMinutes} minutes...`);
  
  // Initial poll
  pollNotionForNewBugs(slackClient);
  
  // Set up interval
  const intervalMs = intervalMinutes * 60 * 1000;
  setInterval(() => {
    pollNotionForNewBugs(slackClient);
  }, intervalMs);
}
