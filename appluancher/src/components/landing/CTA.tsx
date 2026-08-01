import React from 'react';
import Link from 'next/link';
import { ArrowRight, MessageSquare, Mail } from 'lucide-react';

export const CTA: React.FC = () => {
  return (
    <section id="demo" className="py-20" style={{ background: 'linear-gradient(135deg, #165c92 0%, #1a7ab5 100%)' }}>
      <div className="max-w-4xl mx-auto px-4 sm:px-6 lg:px-8 text-center">
        <h2 className="text-3xl md:text-5xl font-bold text-white mb-6">
          Ready to Transform Your Organization?
        </h2>
        <p className="text-lg mb-10 max-w-2xl mx-auto" style={{ color: 'rgba(255, 255, 255, 0.85)' }}>
          Join the organizations using StatGate to turn data into trusted
          evidence and evidence into better decisions.
        </p>

        <div className="flex flex-col sm:flex-row items-center justify-center gap-4">
          <Link
            href="#demo-form"
            className="inline-flex items-center space-x-2 px-6 py-3 bg-white font-medium rounded-md hover:bg-gray-50 transition-colors shadow-lg"
            style={{ color: '#165c92' }}
          >
            <span>Request a Demo</span>
            <ArrowRight className="w-4 h-4" />
          </Link>
          <Link
            href="#contact"
            className="inline-flex items-center space-x-2 px-6 py-3 text-white font-medium rounded-md transition-colors border"
            style={{ background: 'rgba(15, 63, 95, 0.5)', borderColor: 'rgba(255, 255, 255, 0.3)' }}
          >
            <MessageSquare className="w-4 h-4" />
            <span>Talk to Our Team</span>
          </Link>
          <Link
            href="mailto:sales@statgate.ug"
            className="inline-flex items-center space-x-2 px-6 py-3 text-white font-medium rounded-md hover:bg-white/10 transition-colors"
          >
            <Mail className="w-4 h-4" />
            <span>Contact Sales</span>
          </Link>
        </div>
      </div>
    </section>
  );
};