import React from 'react';

const footerLinks = {
  Platform: ['Data Integration', 'Statistical Computing', 'AI Assistants', 'GIS', 'Reporting', 'API'],
  Solutions: ['For Government', 'For Health', 'For Research', 'For Education', 'For Enterprise'],
  Industries: ['Public Health', 'National Statistics', 'Universities', 'NGOs', 'Agriculture'],
  Resources: ['Documentation', 'Case Studies', 'Blog', 'Webinars', 'API Reference'],
  Company: ['About StatGate', 'Careers', 'Contact', 'Partners', 'News'],
};

export const Footer: React.FC = () => {
  return (
    <footer className="bg-gray-900 text-gray-400">
      <div className="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8 py-16">
        <div className="grid grid-cols-2 md:grid-cols-3 lg:grid-cols-6 gap-8">
          {/* Brand */}
          <div className="col-span-2">
            <div className="flex items-center space-x-2 mb-4">
              <img
                src="/icons/logo.png"
                alt="StatGate"
                className="w-8 h-8 rounded-lg"
              />
              <span className="text-lg font-bold text-white">StatGate</span>
            </div>
            <p className="text-sm leading-relaxed max-w-xs">
              The intelligent platform where organizations transform data into
              trusted evidence, and trusted evidence into better decisions.
            </p>
          </div>

          {/* Link columns */}
          {Object.entries(footerLinks).map(([category, links]) => (
            <div key={category}>
              <h4 className="text-sm font-semibold text-white mb-4">
                {category}
              </h4>
              <ul className="space-y-2">
                {links.map((link) => (
                  <li key={link}>
                    <a
                      href="#"
                      className="text-sm hover:text-white transition-colors"
                    >
                      {link}
                    </a>
                  </li>
                ))}
              </ul>
            </div>
          ))}
        </div>

        {/* Bottom bar */}
        <div className="mt-12 pt-8 border-t border-gray-800 flex flex-col md:flex-row items-center justify-between gap-4">
          <p className="text-sm">
            &copy; {new Date().getFullYear()} StatGate. All rights reserved.
          </p>
          <div className="flex items-center space-x-6">
            <a href="#" className="text-sm hover:text-white transition-colors">
              Privacy Policy
            </a>
            <a href="#" className="text-sm hover:text-white transition-colors">
              Terms of Service
            </a>
            <a href="#" className="text-sm hover:text-white transition-colors">
              Cookies
            </a>
          </div>
        </div>
      </div>
    </footer>
  );
};