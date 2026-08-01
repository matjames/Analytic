import React from 'react';
import Link from 'next/link';
import { ArrowRight, Play, Sparkles } from 'lucide-react';

export const Hero: React.FC = () => {
  return (
    <section className="relative pt-20 pb-20 overflow-hidden" style={{ background: 'linear-gradient(135deg, #f9fafb 0%, #f0f7fb 50%, #f9fafb 100%)' }}>
      {/* Background decoration */}
      <div className="absolute inset-0 overflow-hidden pointer-events-none">
        <div className="absolute top-20 left-10 w-72 h-72 rounded-full blur-3xl" style={{ background: 'rgba(22, 92, 146, 0.08)' }} />
        <div className="absolute bottom-10 right-10 w-96 h-96 rounded-full blur-3xl" style={{ background: 'rgba(26, 122, 181, 0.06)' }} />
      </div>

      <div className="relative max-w-7xl mx-auto px-4 sm:px-6 lg:px-8">
        <div className="text-center max-w-4xl mx-auto">
          {/* Badge */}
          <div className="inline-flex items-center space-x-2 px-4 py-1.5 rounded-full mb-6" style={{ background: 'rgba(22, 92, 146, 0.08)', border: '1px solid rgba(22, 92, 146, 0.2)' }}>
            <Sparkles className="w-4 h-4" style={{ color: '#165c92' }} />
            <span className="text-sm font-medium" style={{ color: '#165c92' }}>
              Enterprise Evidence Intelligence Platform
            </span>
          </div>

          {/* Headline */}
          <h1 className="text-4xl md:text-5xl lg:text-6xl font-bold text-gray-900 leading-tight mb-6">
            Transform Data into Evidence.
            <br />
            <span style={{ background: 'linear-gradient(135deg, #165c92 0%, #1a7ab5 100%)', WebkitBackgroundClip: 'text', WebkitTextFillColor: 'transparent', backgroundClip: 'text' }}>
              Transform Evidence into Decisions.
            </span>
          </h1>

          {/* Supporting message */}
          <p className="text-lg md:text-xl text-gray-600 leading-relaxed mb-10 max-w-3xl mx-auto">
            StatGate is an Enterprise Evidence Intelligence Platform that
            unifies data engineering, statistics, artificial intelligence,
            research management, GIS, reporting, and executive decision support
            into one intelligent ecosystem.
          </p>

          {/* CTAs */}
          <div className="flex flex-col sm:flex-row items-center justify-center gap-4">
            <Link
              href="#demo"
              className="inline-flex items-center space-x-2 px-6 py-3 text-white font-medium rounded-md transition-all hover:opacity-90"
              style={{ background: 'linear-gradient(135deg, #165c92 0%, #1a7ab5 100%)', boxShadow: '0 2px 8px rgba(22, 92, 146, 0.25)' }}
            >
              <span>Request a Demo</span>
              <ArrowRight className="w-4 h-4" />
            </Link>
            <Link
              href="#platform"
              className="inline-flex items-center space-x-2 px-6 py-3 bg-white text-gray-700 font-medium rounded-lg border border-gray-200 hover:border-gray-300 hover:bg-gray-50 transition-colors"
            >
              <span>Explore the Platform</span>
            </Link>
            <button className="inline-flex items-center space-x-2 px-6 py-3 text-gray-600 font-medium hover:text-gray-900 transition-colors">
              <Play className="w-4 h-4" />
              <span>Watch Product Overview</span>
            </button>
          </div>
        </div>

        {/* Visual element */}
        <div className="mt-16 relative">
          <div className="max-w-5xl mx-auto bg-white rounded-lg shadow-xl border border-gray-100 overflow-hidden">
            <div className="p-1" style={{ background: 'linear-gradient(135deg, #165c92 0%, #1a7ab5 100%)' }}>
              <div className="bg-white rounded-lg p-6">
                <div className="grid grid-cols-3 md:grid-cols-6 gap-3">
                  {[
                    'Collect',
                    'Integrate',
                    'Validate',
                    'Store',
                    'Analyze',
                    'AI',
                    'Visualize',
                    'Report',
                    'Decide',
                    'Research',
                    'GIS',
                    'Govern',
                  ].map((step, i) => (
                    <div
                      key={step}
                      className="flex flex-col items-center p-3 bg-gray-50 rounded-md border border-gray-100"
                    >
                      <div className="w-8 h-8 rounded-md flex items-center justify-center text-white text-xs font-bold mb-1" style={{ background: 'linear-gradient(135deg, #165c92 0%, #1a7ab5 100%)' }}>
                        {i + 1}
                      </div>
                      <span className="text-xs font-medium text-gray-700">
                        {step}
                      </span>
                    </div>
                  ))}
                </div>
              </div>
            </div>
          </div>
        </div>
      </div>
    </section>
  );
};