import React from 'react';
import {
  Database,
  GitMerge,
  CheckCircle,
  Archive,
  BarChart3,
  Brain,
  Eye,
  FileText,
  Lightbulb,
  ArrowRight,
} from 'lucide-react';

const workflow = [
  { icon: Database, label: 'Collect Data', description: 'Gather data from any source' },
  { icon: GitMerge, label: 'Integrate', description: 'Unify disparate systems' },
  { icon: CheckCircle, label: 'Validate', description: 'Ensure data quality' },
  { icon: Archive, label: 'Store', description: 'Secure enterprise storage' },
  { icon: BarChart3, label: 'Analyze', description: 'Statistical computing' },
  { icon: Brain, label: 'AI', description: 'Intelligent insights' },
  { icon: Eye, label: 'Visualize', description: 'Interactive dashboards' },
  { icon: FileText, label: 'Reports', description: 'Automated reporting' },
  { icon: Lightbulb, label: 'Decisions', description: 'Evidence-based action' },
];

export const Solution: React.FC = () => {
  return (
    <section id="solutions" className="py-20 bg-gradient-to-b from-gray-50 to-white">
      <div className="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8">
        <div className="text-center max-w-3xl mx-auto mb-16">
          <h2 className="text-3xl md:text-4xl font-bold text-gray-900 mb-4">
            The StatGate Solution
          </h2>
          <p className="text-lg text-gray-600">
            StatGate transforms raw data into actionable evidence through an
            end-to-end intelligent workflow.
          </p>
        </div>

        {/* Workflow visualization */}
        <div className="relative">
          <div className="grid grid-cols-3 md:grid-cols-9 gap-3 md:gap-2">
            {workflow.map((step, i) => {
              const Icon = step.icon;
              return (
                <React.Fragment key={step.label}>
                  <div className="flex flex-col items-center text-center">
                    <div className="w-14 h-14 bg-white rounded-xl shadow-md border border-gray-100 flex items-center justify-center mb-2">
                      <Icon className="w-6 h-6" style={{ color: '#165c92' }} />
                    </div>
                    <span className="text-xs font-semibold text-gray-900">
                      {step.label}
                    </span>
                    <span className="text-[10px] text-gray-500 mt-0.5 hidden md:block">
                      {step.description}
                    </span>
                  </div>
                  {i < workflow.length - 1 && (
                    <div className="hidden md:flex items-center justify-center">
                      <ArrowRight className="w-4 h-4 text-gray-300" />
                    </div>
                  )}
                </React.Fragment>
              );
            })}
          </div>
        </div>

        {/* Summary */}
        <div className="mt-16 max-w-3xl mx-auto text-center">
          <p className="text-lg text-gray-700 leading-relaxed">
            From the moment data enters the platform to the moment a leader
            makes a decision, StatGate ensures every step is{' '}
            <span className="font-semibold" style={{ color: '#165c92' }}>intelligent</span>,{' '}
            <span className="font-semibold" style={{ color: '#165c92' }}>validated</span>, and{' '}
            <span className="font-semibold" style={{ color: '#165c92' }}>traceable</span>.
          </p>
        </div>
      </div>
    </section>
  );
};