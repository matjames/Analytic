import React from 'react';
import {
  Calculator,
  FlaskConical,
  Database,
  FileText,
  Activity,
  Map,
  Briefcase,
} from 'lucide-react';

const aiAssistants = [
  {
    icon: Calculator,
    name: 'AI Statistician',
    description:
      'Performs complex statistical analysis, runs hypothesis tests, and interprets results in plain language.',
  },
  {
    icon: FlaskConical,
    name: 'AI Research Assistant',
    description:
      'Helps design studies, manage literature reviews, and structure research methodologies.',
  },
  {
    icon: Database,
    name: 'AI Data Engineer',
    description:
      'Automates data cleaning, transformation, and pipeline construction across multiple sources.',
  },
  {
    icon: FileText,
    name: 'AI Report Writer',
    description:
      'Generates professional reports and summaries from analysis results with customizable templates.',
  },
  {
    icon: Activity,
    name: 'AI Epidemiologist',
    description:
      'Analyzes disease patterns, outbreak trends, and public health data for timely interventions.',
  },
  {
    icon: Map,
    name: 'AI GIS Analyst',
    description:
      'Performs spatial analysis, creates maps, and identifies geographic patterns in your data.',
  },
  {
    icon: Briefcase,
    name: 'AI Policy Advisor',
    description:
      'Translates evidence into policy recommendations and assesses the impact of proposed decisions.',
  },
];

export const AITeam: React.FC = () => {
  return (
    <section id="ai-team" className="py-20 bg-gradient-to-b from-white to-gray-50">
      <div className="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8">
        <div className="text-center max-w-3xl mx-auto mb-16">
          <div className="inline-flex items-center space-x-2 px-4 py-1.5 rounded-full mb-4" style={{ background: 'rgba(22, 92, 146, 0.08)', border: '1px solid rgba(22, 92, 146, 0.2)' }}>
            <span className="text-sm font-medium" style={{ color: '#165c92' }}>
              Powered by Artificial Intelligence
            </span>
          </div>
          <h2 className="text-3xl md:text-4xl font-bold text-gray-900 mb-4">
            Meet Your AI Team
          </h2>
          <p className="text-lg text-gray-600">
            {"StatGate's intelligent assistants are built to accelerate your work across the entire data lifecycle."}
          </p>
        </div>

        <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-6">
          {aiAssistants.map((assistant) => {
            const Icon = assistant.icon;
            return (
              <div
                key={assistant.name}
                className="group p-6 bg-white rounded-xl border border-gray-200 hover:shadow-lg transition-all duration-200"
              >
                <div className="flex items-center space-x-3 mb-3">
                  <div className="w-10 h-10 rounded-lg flex items-center justify-center" style={{ background: 'linear-gradient(135deg, #165c92 0%, #1a7ab5 100%)' }}>
                    <Icon className="w-5 h-5 text-white" />
                  </div>
                  <h3 className="text-base font-semibold text-gray-900">
                    {assistant.name}
                  </h3>
                </div>
                <p className="text-sm text-gray-600 leading-relaxed">
                  {assistant.description}
                </p>
              </div>
            );
          })}
        </div>
      </div>
    </section>
  );
};