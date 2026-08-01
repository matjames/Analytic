import React from 'react';
import {
  Lock,
  ScrollText,
  KeyRound,
  ShieldCheck,
  Activity,
  Code,
  FileCheck,
} from 'lucide-react';

const securityFeatures = [
  {
    icon: Lock,
    title: 'Role-Based Access Control',
    description:
      'Granular permissions ensure users only access what they need.',
  },
  {
    icon: ScrollText,
    title: 'Audit Trails',
    description:
      'Every action is logged for complete traceability and accountability.',
  },
  {
    icon: KeyRound,
    title: 'Encryption',
    description:
      'Data is encrypted at rest and in transit using industry standards.',
  },
  {
    icon: ShieldCheck,
    title: 'Data Governance',
    description:
      'Comprehensive policies for data quality, lineage, and stewardship.',
  },
  {
    icon: Activity,
    title: 'High Availability',
    description:
      'Built for uptime with redundant systems and failover capabilities.',
  },
  {
    icon: Code,
    title: 'Secure APIs',
    description:
      'Authenticated and rate-limited APIs protect your data endpoints.',
  },
  {
    icon: FileCheck,
    title: 'Compliance-Ready',
    description:
      'Architecture designed to support regulatory and audit requirements.',
  },
];

export const Security: React.FC = () => {
  return (
    <section id="resources" className="py-20 bg-gray-900 text-white">
      <div className="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8">
        <div className="text-center max-w-3xl mx-auto mb-16">
          <h2 className="text-3xl md:text-4xl font-bold mb-4">
            Security & Trust
          </h2>
          <p className="text-lg text-gray-300">
            Enterprise-grade security is not an add-on. It is the foundation of
            StatGate.
          </p>
        </div>

        <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-6">
          {securityFeatures.map((feature) => {
            const Icon = feature.icon;
            return (
              <div
                key={feature.title}
                className="p-6 bg-gray-800/50 rounded-xl border border-gray-700 hover:border-gray-600 transition-colors"
              >
                <div className="w-10 h-10 rounded-lg flex items-center justify-center mb-4" style={{ background: 'rgba(26, 122, 181, 0.2)' }}>
                  <Icon className="w-5 h-5" style={{ color: '#1a7ab5' }} />
                </div>
                <h3 className="text-base font-semibold mb-2">
                  {feature.title}
                </h3>
                <p className="text-sm text-gray-400 leading-relaxed">
                  {feature.description}
                </p>
              </div>
            );
          })}
        </div>
      </div>
    </section>
  );
};