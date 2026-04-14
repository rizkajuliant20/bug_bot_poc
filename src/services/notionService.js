import { Client } from '@notionhq/client';
import { config } from '../config.js';

const notion = new Client({
  auth: config.notion.apiKey,
});

export async function createBugTicket(bugData) {
  const {
    title,
    description,
    diagnosis,
    reporter,
    slackThreadUrl,
    threadMessages,
    threadSummary,
  } = bugData;

  try {
    const response = await notion.pages.create({
      parent: {
        database_id: config.notion.databaseId,
      },
      properties: {
        Title: {
          title: [
            {
              text: {
                content: title,
              },
            },
          ],
        },
        Status: {
          status: {
            name: 'Not started',
          },
        },
        Priority: {
          select: {
            name: diagnosis.priority || 'P2',
          },
        },
        Severity: {
          select: {
            name: diagnosis.severity.charAt(0).toUpperCase() + diagnosis.severity.slice(1),
          },
        },
        Category: {
          select: {
            name: diagnosis.category.charAt(0).toUpperCase() + diagnosis.category.slice(1),
          },
        },
        Team: {
          select: {
            name: diagnosis.team ? diagnosis.team.charAt(0).toUpperCase() + diagnosis.team.slice(1) : 'Eng',
          },
        },
        Platform: {
          multi_select: diagnosis.platform ? diagnosis.platform.map(p => ({ name: p.toLowerCase() })) : [{ name: 'web' }],
        },
        Tags: {
          multi_select: diagnosis.tags ? diagnosis.tags.map(t => ({ name: t.toLowerCase() })) : [{ name: 'bug' }],
        },
        Reporter: {
          rich_text: [
            {
              text: {
                content: reporter,
              },
            },
          ],
        },
        'Slack Thread': {
          url: slackThreadUrl,
        },
      },
      children: [
        {
          object: 'block',
          type: 'heading_2',
          heading_2: {
            rich_text: [
              {
                type: 'text',
                text: {
                  content: 'Bug Description',
                },
              },
            ],
          },
        },
        {
          object: 'block',
          type: 'paragraph',
          paragraph: {
            rich_text: [
              {
                type: 'text',
                text: {
                  content: description,
                },
              },
            ],
          },
        },
        {
          object: 'block',
          type: 'heading_2',
          heading_2: {
            rich_text: [
              {
                type: 'text',
                text: {
                  content: 'Precondition',
                },
              },
            ],
          },
        },
        ...(Array.isArray(diagnosis.precondition) && diagnosis.precondition.length > 0
          ? diagnosis.precondition.map(item => ({
              object: 'block',
              type: 'bulleted_list_item',
              bulleted_list_item: {
                rich_text: [
                  {
                    type: 'text',
                    text: {
                      content: item,
                    },
                  },
                ],
              },
            }))
          : [
              {
                object: 'block',
                type: 'paragraph',
                paragraph: {
                  rich_text: [
                    {
                      type: 'text',
                      text: {
                        content: typeof diagnosis.precondition === 'string' ? diagnosis.precondition : 'N/A',
                      },
                    },
                  ],
                },
              },
            ]),
        {
          object: 'block',
          type: 'heading_2',
          heading_2: {
            rich_text: [
              {
                type: 'text',
                text: {
                  content: 'Steps to Reproduce',
                },
              },
            ],
          },
        },
        ...(diagnosis.stepsToReproduce || []).map((step, idx) => ({
          object: 'block',
          type: 'numbered_list_item',
          numbered_list_item: {
            rich_text: [
              {
                type: 'text',
                text: {
                  content: step,
                },
              },
            ],
          },
        })),
        {
          object: 'block',
          type: 'heading_2',
          heading_2: {
            rich_text: [
              {
                type: 'text',
                text: {
                  content: 'Actual Result',
                },
              },
            ],
          },
        },
        ...(Array.isArray(diagnosis.actualResult)
          ? diagnosis.actualResult.map(item => ({
              object: 'block',
              type: 'bulleted_list_item',
              bulleted_list_item: {
                rich_text: [
                  {
                    type: 'text',
                    text: {
                      content: item,
                    },
                  },
                ],
              },
            }))
          : [
              {
                object: 'block',
                type: 'bulleted_list_item',
                bulleted_list_item: {
                  rich_text: [
                    {
                      type: 'text',
                      text: {
                        content: diagnosis.actualResult || 'See description',
                      },
                    },
                  ],
                },
              },
            ]),
        {
          object: 'block',
          type: 'heading_2',
          heading_2: {
            rich_text: [
              {
                type: 'text',
                text: {
                  content: 'Expected Result',
                },
              },
            ],
          },
        },
        ...(Array.isArray(diagnosis.expectedResult)
          ? diagnosis.expectedResult.map(item => ({
              object: 'block',
              type: 'bulleted_list_item',
              bulleted_list_item: {
                rich_text: [
                  {
                    type: 'text',
                    text: {
                      content: item,
                    },
                  },
                ],
              },
            }))
          : [
              {
                object: 'block',
                type: 'bulleted_list_item',
                bulleted_list_item: {
                  rich_text: [
                    {
                      type: 'text',
                      text: {
                        content: diagnosis.expectedResult || 'System should work as intended',
                      },
                    },
                  ],
                },
              },
            ]),
        {
          object: 'block',
          type: 'heading_2',
          heading_2: {
            rich_text: [
              {
                type: 'text',
                text: {
                  content: '🔍 QA Diagnosis',
                },
              },
            ],
          },
        },
        {
          object: 'block',
          type: 'paragraph',
          paragraph: {
            rich_text: [
              {
                type: 'text',
                text: {
                  content: `Root Cause: ${diagnosis.rootCause}`,
                },
                annotations: {
                  bold: true,
                },
              },
            ],
          },
        },
        {
          object: 'block',
          type: 'paragraph',
          paragraph: {
            rich_text: [
              {
                type: 'text',
                text: {
                  content: `Suggested Fix: ${diagnosis.suggestedFix}`,
                },
              },
            ],
          },
        },
        {
          object: 'block',
          type: 'paragraph',
          paragraph: {
            rich_text: [
              {
                type: 'text',
                text: {
                  content: `Affected Components: ${diagnosis.affectedComponents.join(', ')}`,
                },
              },
            ],
          },
        },
        ...(threadSummary
          ? [
              {
                object: 'block',
                type: 'heading_2',
                heading_2: {
                  rich_text: [
                    {
                      type: 'text',
                      text: {
                        content: '💬 Thread Summary',
                      },
                    },
                  ],
                },
              },
              {
                object: 'block',
                type: 'paragraph',
                paragraph: {
                  rich_text: [
                    {
                      type: 'text',
                      text: {
                        content: threadSummary,
                      },
                    },
                  ],
                },
              },
            ]
          : []),
      ],
    });

    return response;
  } catch (error) {
    console.error('Notion API error:', error);
    throw new Error(`Failed to create Notion ticket: ${error.message}`);
  }
}

export async function getNotionPageUrl(pageId) {
  return `https://notion.so/${pageId.replace(/-/g, '')}`;
}
