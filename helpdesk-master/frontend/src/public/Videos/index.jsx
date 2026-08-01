import React from "react";
import './styles.css';

const Videos = () => {

  return (
    <div class="content-area">
      <nav class="breadcrumb-nav">
        <ol class="breadcrumb">
          <li class="breadcrumb-item"><a href="index.html">Operations</a></li>
          <li class="breadcrumb-item active">Training Library</li>
        </ol>
      </nav>

      <div class="page-header">
        <h1 class="page-title">
          <i class="fas fa-play-circle me-2"></i>
          Training Library
        </h1>
        <p class="page-subtitle">Learn how to manage analytics workflows, monitor incidents, and keep reporting reliable.</p>
      </div>

      <div class="search-section">
        <div class="search-bar">
          <input type="text" class="search-input" placeholder="Search training videos..." />
          <i class="fas fa-search search-icon"></i>
        </div>
        <div class="filter-tabs">
          <a href="#" class="filter-tab active">All Videos</a>
          <a href="#" class="filter-tab">Getting Started</a>
          <a href="#" class="filter-tab">Analytics</a>
          <a href="#" class="filter-tab">Workflows</a>
          <a href="#" class="filter-tab">Operations</a>
          <a href="#" class="filter-tab">Reports</a>
        </div>
      </div>

      <div class="search-section">
        <h2 class="section-title">
          <i class="fas fa-star"></i>
          Featured Videos
        </h2>
        <div class="featured-video">
          <div class="featured-thumbnail">
            <div class="play-button">
              <i class="fas fa-play"></i>
            </div>
          </div>
          <div class="featured-content">
            <h3 class="featured-title">Getting started with the StatGate operations workspace</h3>
            <p class="featured-description">A quick tour of the analytics workspace, work queue, and the steps for submitting a new incident.</p>
            <div class="featured-meta">Duration: 12:45 • Views: 2,341 • Updated: 2 days ago</div>
          </div>
        </div>
        <div class="featured-video">
          <div class="featured-thumbnail">
            <div class="play-button">
              <i class="fas fa-play"></i>
            </div>
          </div>
          <div class="featured-content">
            <h3 class="featured-title">How to manage analytics work items</h3>
            <p class="featured-description">Learn how to capture new work items, update progress, and hand off issues to the right team.</p>
            <div class="featured-meta">Duration: 8:32 • Views: 1,892 • Updated: 1 week ago</div>
          </div>
        </div>
      </div>

      <div class="videos-grid">
        <div class="video-card">
          <div class="video-thumbnail">
            <div class="play-button">
              <i class="fas fa-play"></i>
            </div>
            <div class="video-duration">5:23</div>
          </div>
          <div class="video-content">
            <h3 class="video-title">Platform navigation and work queues</h3>
            <p class="video-description">Learn how to move through the operations workspace and manage the work queue.</p>
            <div class="video-meta">
              <span class="video-category">Getting Started</span>
              <span>1,234 views</span>
            </div>
          </div>
        </div>

        <div class="video-card">
          <div class="video-thumbnail">
            <div class="play-button">
              <i class="fas fa-play"></i>
            </div>
            <div class="video-duration">7:45</div>
          </div>
          <div class="video-content">
            <h3 class="video-title">Monitoring dashboard performance</h3>
            <p class="video-description">A practical guide to recognizing delays, refresh failures, and dashboard bottlenecks.</p>
            <div class="video-meta">
              <span class="video-category">Analytics</span>
              <span>987 views</span>
            </div>
          </div>
        </div>

        <div class="video-card">
          <div class="video-thumbnail">
            <div class="play-button">
              <i class="fas fa-play"></i>
            </div>
            <div class="video-duration">4:12</div>
          </div>
          <div class="video-content">
            <h3 class="video-title">Creating and updating work items</h3>
            <p class="video-description">Step-by-step process for recording a new work item and keeping updates visible.</p>
            <div class="video-meta">
              <span class="video-category">Workflows</span>
              <span>756 views</span>
            </div>
          </div>
        </div>

        <div class="video-card">
          <div class="video-thumbnail">
            <div class="play-button">
              <i class="fas fa-play"></i>
            </div>
            <div class="video-duration">9:18</div>
          </div>
          <div class="video-content">
            <h3 class="video-title">Access and onboarding workflow</h3>
            <p class="video-description">How to request and approve access for new analysts and team members.</p>
            <div class="video-meta">
              <span class="video-category">Operations</span>
              <span>543 views</span>
            </div>
          </div>
        </div>

        <div class="video-card">
          <div class="video-thumbnail">
            <div class="play-button">
              <i class="fas fa-play"></i>
            </div>
            <div class="video-duration">6:34</div>
          </div>
          <div class="video-content">
            <h3 class="video-title">Generating executive reports</h3>
            <p class="video-description">Learn how to prepare and publish consistent reports for leadership review.</p>
            <div class="video-meta">
              <span class="video-category">Reports</span>
              <span>432 views</span>
            </div>
          </div>
        </div>
      </div>
    </div>
  );
};

export default Videos;
