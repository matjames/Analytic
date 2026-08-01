import React from 'react';
import {
  Database,
  FileText,
  AlertTriangle,
  Clock,
  BarChart3,
  Unlink,
  Brain,
} from 'lucide-react';

const problems = [
  {
    icon: Database,
    title: 'Data Scattered Across Systems',
    description:
      'Critical data lives in silos across multiple platforms, making it impossible to get a unified view.',
  },
  {
    icon: FileText,
    title: 'Manual Reporting',
    description:
      'Teams waste hours on manual report generation instead of focusing on analysis and insights.',
  },
  {
    icon: AlertTriangle,
    title: 'Poor Data Quality',
    description:
      'Inconsistent, incomplete, and unvalidated data leads to unreliable conclusions.',
  },
  {
    icon: Clock,
    title: 'Slow Decision-Making',
    description:
      'Without real-time insights, organizations react too late to emerging trends and challenges.',
  },
  {
    icon: BarChart3,
    title: 'Limited Analytics',
    description:
      'Basic dashboards fail to uncover the deep patterns and relationships hidden in your data.',
  },
  {
    icon: Unlink,
    title: 'Disconnected Platforms',
    description:
      'Teams use separate tools for data, analytics, research, and reporting with no integration.',
  },
  {
    icon: Brain,
    title: 'Underutilized AI',
    description:
      'Artificial intelligence remains a buzzword rather than a practical tool driving real outcomes.',
  },
];

export const Problem: React.FC = () => {
  return (
    <section className="py-20 bg-white">
      <div className="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8">
        <div className="text-center max-w-3xl mx-auto mb-16">
          <h2 className="text-3xl md:text-4xl font-bold text-gray-900 mb-4">
            The Challenge Facing Modern Organizations
          </h2>
          <p className="text-lg text-gray-600">
            Organizations today struggle to turn raw data into trusted
            evidence. The gaps are everywhere.
          </p>
        </div>

        <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-6">
          {problems.map((problem) => {
            const Icon = problem.icon;
            return (
              <div
                key={problem.title}
                className="p-6 bg-gray-50 rounded-xl border border-gray-100 hover:border-gray-200 transition-colors"
              >
                <div className="w-10 h-10 bg-red-50 rounded-lg flex items-center justify-center mb-4">
                  <Icon className="w-5 h-5 text-red-500" />
                </div>
                <h3 className="text-base font-semibold text-gray-900 mb-2">
                  {problem.title}
                </h3>
                <p className="text-sm text-gray-600 leading-relaxed">
                  {problem.description}
                </p>
              </div>
            );
          })}
        </div>

        {/* Transition to solution */}
        <div className="mt-16 text-center">
          <div className="inline-block px-6 py-3 rounded-full" style={{ background: 'linear-gradient(135deg, #165c92 0%, #1a7ab5 100%)' }}>
            <p className="text-white font-medium">
              StatGate is the unified solution.
            </p>
          </div>
        </div>
      </div>
    </section>
  );
};