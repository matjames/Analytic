import React from 'react';
import {
  Landmark,
  HeartPulse,
  BarChart3,
  GraduationCap,
  FlaskConical,
  Globe,
  HeartHandshake,
  Banknote,
  Wheat,
  BookOpen,
  Building2,
} from 'lucide-react';

const industries = [
  { icon: Landmark, name: 'Government' },
  { icon: HeartPulse, name: 'Public Health' },
  { icon: BarChart3, name: 'National Statistics Offices' },
  { icon: GraduationCap, name: 'Universities' },
  { icon: FlaskConical, name: 'Research Institutions' },
  { icon: Globe, name: 'NGOs' },
  { icon: HeartHandshake, name: 'Development Partners' },
  { icon: Banknote, name: 'Financial Institutions' },
  { icon: Wheat, name: 'Agriculture' },
  { icon: BookOpen, name: 'Education' },
  { icon: Building2, name: 'Private Enterprises' },
];

export const Industries: React.FC = () => {
  return (
    <section id="industries" className="py-20 bg-gray-50">
      <div className="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8">
        <div className="text-center max-w-3xl mx-auto mb-16">
          <h2 className="text-3xl md:text-4xl font-bold text-gray-900 mb-4">
            Industries We Serve
          </h2>
          <p className="text-lg text-gray-600">
            StatGate is built for organizations that rely on data-driven
            decision-making across diverse sectors.
          </p>
        </div>

        <div className="grid grid-cols-2 md:grid-cols-3 lg:grid-cols-4 gap-4">
          {industries.map((industry) => {
            const Icon = industry.icon;
            return (
              <div
                key={industry.name}
                className="flex items-center space-x-3 p-4 bg-white rounded-xl border border-gray-100 hover:border-blue-200 hover:shadow-sm transition-all"
              >
                <div className="w-10 h-10 rounded-lg flex items-center justify-center flex-shrink-0" style={{ background: 'rgba(22, 92, 146, 0.08)' }}>
                  <Icon className="w-5 h-5" style={{ color: '#165c92' }} />
                </div>
                <span className="text-sm font-medium text-gray-900">
                  {industry.name}
                </span>
              </div>
            );
          })}
        </div>
      </div>
    </section>
  );
};