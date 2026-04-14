import OpenAI from 'openai';
import { config } from '../config.js';
import { logInfo, logSuccess, logError, logFlow } from '../utils/logger.js';

const openai = new OpenAI({
  apiKey: config.openai.apiKey,
});

export async function diagnoseBug(bugDescription, threadMessages = []) {
  logFlow('AI_SERVICE', 'Starting bug diagnosis', { 
    descriptionLength: bugDescription.length, 
    threadMessageCount: threadMessages.length 
  });
  
  const context = threadMessages.length > 0
    ? `\n\nThread context:\n${threadMessages.map(m => `- ${m.user}: ${m.text}`).join('\n')}`
    : '';

  const prompt = `You are a technical bug analyst. Analyze the following bug report and extract structured information.

Bug Report:
${bugDescription}${context}

Provide your analysis in the following JSON format:
{
  "severity": "critical|high|medium|low",
  "category": "backend|frontend|database|api|ui/ux|performance|security|other",
  "priority": "P0|P1|P2|P3",
  "platform": ["ios", "android", "web", "backend"],
  "team": "eng|data|ui/ux|marketing|design|product|",
  "precondition": "What conditions must be met before the bug occurs (bullet points)",
  "stepsToReproduce": ["Step 1", "Step 2", "Step 3"],
  "actualResult": "What actually happens",
  "expectedResult": "What should happen",
  "rootCause": "Brief analysis of the likely root cause",
  "suggestedFix": "Recommended solution or next steps",
  "affectedComponents": ["component1", "component2"],
  "tags": ["bug", "feature", "enhancement", etc]
}

Extract as much structured information as possible from the bug report. If information is missing, use reasonable defaults.`;

  try {
    logInfo('Calling OpenAI API for diagnosis');
    const startTime = Date.now();
    const response = await openai.chat.completions.create({
      model: 'gpt-3.5-turbo',
      messages: [
        {
          role: 'system',
          content: 'You are a technical bug analyst. Always respond with valid JSON only.',
        },
        {
          role: 'user',
          content: prompt,
        },
      ],
      temperature: 0.3,
      response_format: { type: 'json_object' },
    });

    const diagnosis = JSON.parse(response.choices[0].message.content);
    const duration = Date.now() - startTime;
    logSuccess('Bug diagnosis completed', { 
      duration: `${duration}ms`, 
      severity: diagnosis.severity, 
      category: diagnosis.category 
    });
    return diagnosis;
  } catch (error) {
    logError('AI diagnosis failed', error, { descriptionLength: bugDescription.length });
    return {
      severity: 'medium',
      category: 'other',
      priority: 'P2',
      platform: ['web'],
      team: 'eng',
      precondition: 'Unable to extract automatically',
      stepsToReproduce: ['See bug description'],
      actualResult: 'See bug description',
      expectedResult: 'System should work as intended',
      rootCause: 'Unable to analyze automatically',
      suggestedFix: 'Manual investigation required',
      affectedComponents: ['unknown'],
      tags: ['bug'],
    };
  }
}

export async function summarizeThread(threadMessages) {
  if (threadMessages.length <= 1) {
    logInfo('Skipping thread summary - insufficient messages', { count: threadMessages.length });
    return null;
  }
  
  logFlow('AI_SERVICE', 'Starting thread summarization', { messageCount: threadMessages.length });

  const conversation = threadMessages
    .map(m => `${m.user}: ${m.text}`)
    .join('\n');

  const prompt = `Summarize the following bug discussion thread. Extract:
1. Key points discussed
2. Additional context or symptoms mentioned
3. Any attempted solutions or workarounds

Thread:
${conversation}

Provide a concise summary (max 200 words) focusing on technical details.`;

  try {
    logInfo('Calling OpenAI API for thread summary');
    const startTime = Date.now();
    const response = await openai.chat.completions.create({
      model: 'gpt-3.5-turbo',
      messages: [
        {
          role: 'system',
          content: 'You are a technical writer. Summarize bug discussions concisely.',
        },
        {
          role: 'user',
          content: prompt,
        },
      ],
      temperature: 0.3,
      max_tokens: 300,
    });

    const summary = response.choices[0].message.content.trim();
    const duration = Date.now() - startTime;
    logSuccess('Thread summary completed', { duration: `${duration}ms`, summaryLength: summary.length });
    return summary;
  } catch (error) {
    logError('Thread summarization failed', error, { messageCount: threadMessages.length });
    return null;
  }
}

export async function generateBugSummary(bugDescription, diagnosis, threadMessages = []) {
  logFlow('AI_SERVICE', 'Generating bug title', { 
    descriptionLength: bugDescription.length,
    threadMessageCount: threadMessages.length 
  });
  
  let appName = 'App';
  let detectedAppName = null;
  
  const allText = [
    bugDescription,
    ...threadMessages.map(m => m.text || '')
  ].join(' ').toLowerCase();
  
  if (allText.includes('jago app') || allText.includes('jagoapp')) {
    appName = 'Jago App';
    detectedAppName = 'Jago App';
    logInfo('Detected app name from text', { appName: 'Jago App' });
  } else if (allText.includes('jagoan app') || allText.includes('jagoanapp')) {
    appName = 'Jagoan App';
    detectedAppName = 'Jagoan App';
    logInfo('Detected app name from text', { appName: 'Jagoan App' });
  } else if (allText.includes('depot portal') || allText.includes('depot')) {
    appName = 'Depot Portal';
    detectedAppName = 'Depot Portal';
  } else if (allText.includes('service') || allText.includes('backend') || allText.includes('api')) {
    appName = 'Service';
    detectedAppName = 'Service';
  } else if (diagnosis.platform && diagnosis.platform.includes('android')) {
    appName = 'Jagoan App';
    detectedAppName = 'Jagoan App';
  } else if (diagnosis.platform && diagnosis.platform.includes('ios')) {
    appName = 'Jagoan App';
    detectedAppName = 'Jagoan App';
  }

  const prompt = `Create a concise bug ticket title in this exact format:
[Bug][${appName}] Brief description of the issue

Bug: ${bugDescription}
Category: ${diagnosis.category}

Keep the description brief and clear (max 8 words after app name). Focus on the core issue.`;

  try {
    logInfo('Calling OpenAI API for bug title generation', { appName });
    const startTime = Date.now();
    const response = await openai.chat.completions.create({
      model: 'gpt-3.5-turbo',
      messages: [
        {
          role: 'system',
          content: 'You are a technical writer. Create clear, structured bug titles following the exact format provided.',
        },
        {
          role: 'user',
          content: prompt,
        },
      ],
      temperature: 0.3,
      max_tokens: 60,
    });

    const title = response.choices[0].message.content.trim();
    const duration = Date.now() - startTime;
    logSuccess('Bug title generated', { duration: `${duration}ms`, title });
    return { title, appName: detectedAppName };
  } catch (error) {
    logError('Bug title generation failed', error);
    return { title: `[Bug][App] ${bugDescription.substring(0, 50)}...`, appName: detectedAppName };
  }
}
