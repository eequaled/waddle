import { Session } from '../types';

export const MOCK_SESSIONS: Session[] = [
  {
    id: '1',
    title: 'Researching WebSockets & Real-time Data',
    summary: 'You explored several WebSocket libraries on GitHub and read documentation for Socket.io. You then started a new project in VS Code and pasted some example code. You briefly discussed implementation details on Slack with Sarah.',
    tags: ['dev', 'websockets', 'research'],
    startTime: '2:30 PM',
    endTime: '4:15 PM',
    duration: '1h 45m',
    apps: ['Chrome', 'VS Code', 'Slack'],
    date: 'Today',
    activities: [
      { id: 'a1', app: 'Chrome', description: 'Viewed "Socket.io Documentation"', timestamp: '2:30 PM' },
      { id: 'a2', app: 'Chrome', description: 'Viewed "GitHub - uWebSockets/uWebSockets"', timestamp: '2:45 PM' },
      { id: 'a3', app: 'VS Code', description: 'Created new file "server.js"', timestamp: '3:00 PM' },
      { id: 'a4', app: 'Slack', description: 'Message to @Sarah: "Socket.io looks promising"', timestamp: '3:15 PM' }
    ],
    content: [
      { id: 'b0', type: 'summary', content: 'You explored several WebSocket libraries on GitHub and read documentation for Socket.io. You then started a new project in VS Code and pasted some example code. You briefly discussed implementation details on Slack with Sarah.' },
      { id: 'b1', type: 'heading', content: 'Key Takeaways' },
      { id: 'b2', type: 'todo', content: 'Socket.io simplifies connection handling', checked: true },
      { id: 'b3', type: 'todo', content: 'Need to consider scaling implications', checked: false },
      { id: 'b4', type: 'code', content: `const default, javaSchte, () => {
  const chat() => {
    consumerMessage = nosdind('setEndSyring') => {
      console.log(tnase)
      console.log(print)
    }
  }
}`, language: 'javascript' }
    ]
  },
  {
    id: '2',
    title: 'Budget Planning',
    summary: 'You reviewed the Q3 budget spreadsheet and compared pricing on Stripe. You then updated the financial projections and messaged the finance team.',
    tags: ['finance', 'planning'],
    startTime: '1:00 PM',
    endTime: '2:00 PM',
    duration: '1h',
    apps: ['Chrome', 'Excel', 'Slack'],
    date: 'Today',
    activities: [
      { id: 'a5', app: 'Chrome', description: 'Viewed "Stripe Pricing Page"', timestamp: '1:00 PM' },
      { id: 'a6', app: 'Excel', description: 'Opened "Q3_budget.xlsx"', timestamp: '1:10 PM' },
      { id: 'a7', app: 'Slack', description: 'Sent message to @finance', timestamp: '1:50 PM' }
    ],
    content: [
      { id: 'b5', type: 'summary', content: 'You reviewed the Q3 budget spreadsheet and compared pricing on Stripe. You then updated the financial projections and messaged the finance team.' },
      { id: 'b6', type: 'heading', content: 'Budget Adjustments' },
      { id: 'b7', type: 'paragraph', content: 'Need to allocate more funds for the server infrastructure based on the new WebSocket requirements.' }
    ]
  },
  {
    id: '3',
    title: 'Designing new landing page hero section',
    summary: 'Focused design session in Figma working on the hero section iterations. Listened to Spotify throughout.',
    tags: ['design', 'focus mode'],
    startTime: '11:00 AM',
    endTime: '12:30 PM',
    duration: '1.5h',
    apps: ['Figma', 'Spotify'],
    date: 'Today',
    activities: [
      { id: 'a8', app: 'Figma', description: 'Edited "Landing Page V2"', timestamp: '11:00 AM' },
      { id: 'a9', app: 'Spotify', description: 'Played "Deep Focus Playlist"', timestamp: '11:05 AM' }
    ],
    content: [
      { id: 'b8', type: 'summary', content: 'Focused design session in Figma working on the hero section iterations. Listened to Spotify throughout.' },
      { id: 'b9', type: 'heading', content: 'Hero Iterations' },
      { id: 'b10', type: 'paragraph', content: 'Option A is cleaner, but Option B has better CTA visibility.' },
      { id: 'b_img', type: 'image', content: 'placeholder' }
    ]
  },
  {
    id: '4',
    title: 'Planning Summer Vacation & Budgeting',
    summary: 'You spent the last hour and a half researching travel options for Japan. You looked at flights on Google Flights and cross-referenced hotels on Tripadvisor.',
    tags: ['research', 'finance'],
    startTime: 'Yesterday',
    endTime: '2:00 PM',
    duration: '1.5h',
    apps: ['Chrome', 'Slack', 'Notes'],
    date: 'Yesterday',
    activities: [
      { id: 'a11', app: 'Chrome', description: 'Viewed "Top 10 Hotels in Kyoto"', timestamp: '2:00 PM' },
      { id: 'a12', app: 'Chrome', description: 'Viewed "Google Flights: LAX to KIX"', timestamp: '2:15 PM' },
      { id: 'a13', app: 'Slack', description: 'Message to @Sarah: "Found some flight options"', timestamp: '2:45 PM' },
      { id: 'a14', app: 'Notes', description: 'Created new note: "Japan Itinerary Draft"', timestamp: '3:00 PM' }
    ],
    content: [
      { id: 'b11', type: 'summary', content: 'You spent the last hour and a half researching travel options for Japan. You looked at flights on Google Flights and cross-referenced hotels on Tripadvisor.' },
      { id: 'b12', type: 'heading', content: 'Flight Options' },
      { id: 'b13', type: 'paragraph', content: '$1,200 round trip estimate via ANA.' }
    ]
  }
];
