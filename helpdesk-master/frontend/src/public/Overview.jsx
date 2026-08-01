import React from 'react';
import { Link } from 'react-router-dom';

const Overview = () => {
  return (
    <div className="container py-4">
      <div className="row g-4">
        <div className="col-lg-8">
          <div className="card border-0 shadow-sm p-4">
            <h2 className="mb-3">Operations Console Activated</h2>
            <p className="text-muted mb-3">
              The StatGate operations workspace is now active and ready for incident intake, work queue review,
              operational knowledge, and guided response workflows.
            </p>
            <div className="d-flex flex-wrap gap-2">
              <Link to="/public/tickets" className="btn btn-primary">Open work queue</Link>
              <Link to="/public/knowledgeBase" className="btn btn-outline-secondary">Knowledge base</Link>
              <Link to="/admin/dashboard" className="btn btn-outline-secondary">Admin workspace</Link>
            </div>
          </div>
        </div>

        <div className="col-lg-4">
          <div className="card border-0 shadow-sm p-4 h-100">
            <h5 className="mb-3">Operations modules</h5>
            <ul className="list-unstyled mb-0">
              <li className="mb-2">• Incident and request intake</li>
              <li className="mb-2">• Guided workflows and triage</li>
              <li className="mb-2">• Knowledge base and training library</li>
              <li className="mb-2">• Admin analytics and operations workspace</li>
            </ul>
          </div>
        </div>
      </div>
    </div>
  );
};

export default Overview;
