import React from 'react';
import {
  Layers,
  Server,
  Brain,
  ShieldCheck,
  Code,
  Users,
  Database,
  FlaskConical,
} from 'lucide-react';

const reasons = [
  {
    icon: Layers,
    title: 'One Unified Platform',
    description:
      'No more switching between disconnected tools. StatGate brings everything together.',
  },
  {
    icon: Server,
    title: 'Enterprise-Ready Architecture',
    description:
      'Built to scale from small teams to national-level deployments with high availability.',
  },
  {
    icon: Brain,
    title: 'AI-Powered Insights',
    description:
      'Intelligent assistants accelerate analysis and surface insights you would miss.',
  },
  {
    icon: ShieldCheck,
    title: 'Secure and Scalable',
    description:
      'Enterprise-grade security with encryption, access control, and audit capabilities.',
  },
  {
    icon: Code,
    title: 'API-First Design',
    description:
      'Integrate StatGate with your existing systems through a comprehensive REST API.',
  },
  {
    icon: Users,
    title: 'Role-Based Workspaces',
    description:
      'Every user gets a tailored experience based on their role and responsibilities.',
  },
  {
    icon: Database,
    title: 'End-to-End Data Lifecycle',
    description:
      'From collection to decision, manage the entire data journey in one place.',
  },
  {
    icon: FlaskConical,
    title: 'Research and Analytics Together',
    description:
      'Combine rigorous research methodologies with powerful analytics in one ecosystem.',
  },
];

export const WhyChoose: React.FC = () => {
  return (
    <section id="company" className="py-20 bg-white">
      <div className="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8">
        <div className="text-center max-w-3xl mx-auto mb-16">
          <h2 className="text-3xl md:text-4xl font-bold text-gray-900 mb-4">
            Why Choose StatGate
          </h2>
          <p className="text-lg text-gray-600">
            StatGate is more than software. It is the infrastructure for
            evidence-driven organizations.
          </p>
        </div>

        <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-4 gap-6">
          {reasons.map((reason) => {
            const Icon = reason.icon;
            return (
              <div key={reason.title} className="text-center">
                <div className="w-14 h-14 rounded-xl flex items-center justify-center mx-auto mb-4" style={{ background: 'linear-gradient(135deg, #165c92 0%, #1a7ab5 100%)' }}>
                  <Icon className="w-7 h-7 text-white" />
                </div>
                <h3 className="text-base font-semibold text-gray-900 mb-2">
                  {reason.title}
                </h3>
                <p className="text-sm text-gray-600 leading-relaxed">
                  {reason.description}
                </p>
              </div>
            );
          })}
        </div>
      </div>
    </section>
  );
};