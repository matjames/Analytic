import React from 'react';
import {
  GitMerge,
  Calculator,
  Brain,
  FlaskConical,
  Map,
  ShieldCheck,
  LayoutDashboard,
  Lightbulb,
  Lock,
  Code,
} from 'lucide-react';

const capabilities = [
  {
    icon: GitMerge,
    title: 'Data Integration',
    description:
      'Connect and unify data from databases, APIs, files, and external systems into a single source of truth.',
  },
  {
    icon: Calculator,
    title: 'Statistical Computing',
    description:
      'Perform advanced statistical analysis, modeling, and hypothesis testing with enterprise-grade tools.',
  },
  {
    icon: Brain,
    title: 'Artificial Intelligence',
    description:
      'Leverage AI assistants and machine learning models to uncover patterns and generate intelligent insights.',
  },
  {
    icon: FlaskConical,
    title: 'Research Management',
    description:
      'Manage research projects, studies, and experiments with full traceability and collaboration tools.',
  },
  {
    icon: Map,
    title: 'GIS & Spatial Intelligence',
    description:
      'Visualize and analyze geographic data with powerful mapping and spatial analytics capabilities.',
  },
  {
    icon: ShieldCheck,
    title: 'Metadata & Data Governance',
    description:
      'Maintain data lineage, quality standards, and governance policies across the entire platform.',
  },
  {
    icon: LayoutDashboard,
    title: 'Dashboards & Reporting',
    description:
      'Create interactive dashboards and automated reports that communicate insights clearly and effectively.',
  },
  {
    icon: Lightbulb,
    title: 'Decision Support',
    description:
      'Empower leaders with evidence-based recommendations and scenario analysis for confident decision-making.',
  },
  {
    icon: Lock,
    title: 'Enterprise Security',
    description:
      'Role-based access control, audit trails, encryption, and compliance-ready architecture built in.',
  },
  {
    icon: Code,
    title: 'API & Integrations',
    description:
      'Extend the platform with a comprehensive API and integrate with your existing technology stack.',
  },
];

export const Capabilities: React.FC = () => {
  return (
    <section id="platform" className="py-20 bg-white">
      <div className="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8">
        <div className="text-center max-w-3xl mx-auto mb-16">
          <h2 className="text-3xl md:text-4xl font-bold text-gray-900 mb-4">
            Platform Capabilities
          </h2>
          <p className="text-lg text-gray-600">
            Everything your organization needs to manage the full data
            lifecycle in one unified platform.
          </p>
        </div>

        <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-6">
          {capabilities.map((capability) => {
            const Icon = capability.icon;
            return (
              <div
                key={capability.title}
                className="group p-6 bg-white rounded-xl border border-gray-200 hover:border-blue-200 hover:shadow-lg transition-all duration-200"
              >
                <div className="w-12 h-12 rounded-xl flex items-center justify-center mb-4 transition-colors" style={{ background: 'rgba(22, 92, 146, 0.08)' }}>
                  <Icon className="w-6 h-6" style={{ color: '#165c92' }} />
                </div>
                <h3 className="text-base font-semibold text-gray-900 mb-2">
                  {capability.title}
                </h3>
                <p className="text-sm text-gray-600 leading-relaxed">
                  {capability.description}
                </p>
              </div>
            );
          })}
        </div>
      </div>
    </section>
  );
};