export async function getThreadMessages(client, channel, threadTs) {
  try {
    const result = await client.conversations.replies({
      channel: channel,
      ts: threadTs,
    });

    return result.messages.map(msg => ({
      user: msg.user || 'Unknown',
      text: msg.text || '',
      ts: msg.ts,
    }));
  } catch (error) {
    console.error('Error fetching thread messages:', error);
    return [];
  }
}

export async function getUserInfo(client, userId) {
  try {
    const result = await client.users.info({
      user: userId,
    });
    return result.user.real_name || result.user.name || 'Unknown User';
  } catch (error) {
    console.error('Error fetching user info:', error);
    return 'Unknown User';
  }
}

export function getSlackThreadUrl(teamId, channel, threadTs) {
  const messageId = 'p' + threadTs.replace(/\./g, '');
  return `https://slack.com/archives/${channel}/${messageId}`;
}

export async function sendThreadReply(client, channel, threadTs, text, blocks = null) {
  try {
    await client.chat.postMessage({
      channel: channel,
      thread_ts: threadTs,
      text: text,
      blocks: blocks,
    });
  } catch (error) {
    console.error('Error sending thread reply:', error);
  }
}
