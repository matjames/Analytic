# StatChat Workplan

Current milestone: 1 — Core Enterprise Messaging Platform
Status: in progress

This workplan breaks the StatChat directive into traceable, implementation-ready tasks.

## Milestone 1: Core Enterprise Messaging Platform

1. Define core architecture and integration points
   - [x] Document module boundaries for StatChat vs StatGate modules
   - [x] Define launcher-based authentication flow and current user bootstrap
   - [x] Define API contract for Identity & Access, Organizations, Users, Roles & Permissions
   - [x] Define technology stack for backend (Go), frontend (React + TypeScript), and real-time layer

2. Build fundamental messaging services
   - [x] Create message model and persistence design
   - [x] Implement REST APIs for conversations, channels, messages
   - [x] Implement WebSocket endpoint for real-time messaging
   - [ ] Add support for direct messages and channel messages
   - [x] Implement channel list retrieval and message history retrieval

3. Build professional frontend experience
   - [ ] Create module shell with StatChat navigation
   - [ ] Implement three-panel chat layout
   - [ ] Implement channel navigation and message rendering
   - [x] Add placeholder login UI for launcher-managed auth
   - [ ] Ensure responsive layout and mobile-friendly behavior
   - [x] Add dark/light theme support scaffold

4. Validate core acceptance criteria
   - [ ] Users can exchange real-time messages
   - [ ] Teams and channels render correctly
   - [ ] StatChat is launcher-integrated with no local login required

## Milestone 2: Enterprise Collaboration Features

5. Direct messaging and chat polishing
   - [ ] Add emojis, reactions, and message formatting support
   - [ ] Add message editing and message deletion
   - [ ] Add read receipts and typing indicators
   - [ ] Add online status and presence tracking
   - [ ] Add pinned messages and bookmarks
   - [ ] Add message search by conversation

6. Group chat and channel management
   - [ ] Implement group creation, description, avatar, and admin roles
   - [ ] Implement public/private/invite-only channel permissions
   - [ ] Add channel membership management and group members UI
   - [ ] Add channel discovery and favorites

7. File sharing and attachments
   - [ ] Implement file upload service
   - [ ] Support attachments in messages
   - [ ] Add preview support for documents, images, and media where possible
   - [ ] Add file metadata and permissions enforcement

## Milestone 3: Meetings, Calls, and Live Collaboration

8. Meetings system
   - [ ] Implement meeting scheduling APIs and data model
   - [ ] Implement instant meeting creation
   - [ ] Add meeting agenda, attendees, notes, and action items
   - [ ] Add calendar integration and upcoming meeting list

9. Voice and video calls
   - [ ] Implement one-to-one voice call flow
   - [ ] Implement group call flow
   - [ ] Add mute, hold, transfer, and recording metadata support
   - [ ] Add HD video, screen sharing, and participant management
   - [ ] Add waiting room and breakout room support scaffold

## Milestone 4: Project, Research, and Knowledge Integration

10. Research spaces and project collaboration
    - [ ] Automatically create research space per project
    - [ ] Add collaboration space structure: chat, members, meetings, files, datasets, reports, tasks
    - [ ] Implement project discussion and shared calendar
    - [ ] Implement milestones, activity log, and project-level notifications

11. Dataset and report discussions
    - [ ] Add dataset discussion threads for comments, reviews, approvals, and version conversations
    - [ ] Add report discussion with comments, approval workflow, revision history, and mentions

12. Task management and workflow
    - [ ] Implement task model with title, description, assignee, priority, due date, status, attachments, and discussion
    - [ ] Enable task creation from conversations
    - [ ] Add task list and task detail UI

## Milestone 5: Search, Notifications, and Presence

13. Notifications and alerts
    - [ ] Implement in-app notifications
    - [ ] Implement desktop notifications integration
    - [ ] Add email notification hooks (SMS future)
    - [ ] Notify on new messages, mentions, meetings, tasks, approvals, dataset updates

14. Global search
    - [ ] Implement search index for users, messages, files, channels, meetings, reports, datasets, projects, tasks
    - [ ] Add search UI and result navigation

15. User presence and status
    - [ ] Implement user presence statuses: online, busy, away, offline, in meeting, presenting, do not disturb
    - [ ] Add presence indicators throughout the UI

## Milestone 6: Security, Scalability, and Testing

16. Security and compliance
    - [ ] Enforce TLS for transport
    - [ ] Implement role-based permissions and organization isolation
    - [ ] Add audit logs, activity logs, and retention policy hooks
    - [ ] Add soft delete support and file permission enforcement

17. Performance and scale
    - [ ] Design message persistence for large scale and indexing
    - [ ] Implement efficient media upload handling
    - [ ] Add optimistic UI updates and background synchronization
    - [ ] Design for thousands of concurrent users

18. Testing and acceptance
    - [ ] Add unit tests for backend services
    - [ ] Add integration tests for API and WebSocket flows
    - [ ] Add load tests for real-time and media paths
    - [ ] Add security tests for auth, permissions, and transport
    - [ ] Add UI tests for chat flows and responsive behavior
    - [ ] Add cross-browser compatibility checks

## Traceability and documentation

19. Document the platform
    - [ ] Create OpenAPI/Swagger documentation for all REST APIs
    - [ ] Document WebSocket endpoints and real-time events
    - [ ] Document data model and database schema
    - [ ] Record feature dependencies and module integration points

20. Release readiness
    - [ ] Verify all StatGate module integrations: Research, Projects, Reports, Datasets, Tasks, Workflow Engine, Document Management, Notifications
    - [ ] Validate acceptance criteria end-to-end
    - [ ] Prepare deployment checklist for enterprise readiness
