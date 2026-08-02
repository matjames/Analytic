import { useState } from 'react';
import styles from './LegalPanel.module.css';

export type LegalTab = 'privacy' | 'terms' | 'status';

interface Props {
  open: boolean;
  initialTab?: LegalTab;
  onClose: () => void;
}

interface TabMeta {
  id: LegalTab;
  label: string;
}

const tabs: TabMeta[] = [
  { id: 'privacy', label: 'Privacy Policy' },
  { id: 'terms', label: 'Terms of Service' },
  { id: 'status', label: 'System Status' },
];

const serviceStatuses = [
  { name: 'StatChat Messaging', uptime: '99.99%', operational: true },
  { name: 'WebSocket Gateway', uptime: '99.98%', operational: true },
  { name: 'Push Notifications', uptime: '99.96%', operational: true },
  { name: 'File & Media Transfer', uptime: '99.97%', operational: true },
  { name: 'Calendar & Meetings', uptime: '99.95%', operational: true },
  { name: 'Search Indexer', uptime: '99.93%', operational: true },
  { name: 'Authentication Service', uptime: '99.99%', operational: true },
  { name: 'Data Warehouse Sync', uptime: '99.91%', operational: true },
];

function PrivacyContent() {
  return (
    <>
      <h2>Privacy Policy</h2>
      <p>
        StatGate is committed to protecting your privacy. This Privacy Policy explains how we collect,
        use, disclose, and safeguard your information when you use StatChat, the enterprise communication
        and collaboration platform (&ldquo;the Service&rdquo;).
      </p>

      <h3>Information We Collect</h3>
      <p>
        <strong>Account Information:</strong> When you register or are provisioned for an account, we collect
        your name, email address, organization affiliation, and role assignments. Profile photographs and
        optional &ldquo;About&rdquo; descriptions are collected when you choose to provide them.
      </p>
      <p>
        <strong>Communication Content:</strong> StatChat processes the messages, files, voice notes, calendar
        invitations, and meeting recordings you create or share through the Service. This content is stored
        encrypted at rest within your organization&rsquo;s dedicated data partition.
      </p>
      <p>
        <strong>Usage Data:</strong> We collect diagnostic and usage data&mdash;such as feature interaction logs,
        session duration, and error telemetry&mdash;solely for the purpose of improving service reliability and
        performance. This data is pseudonymised and never sold to third parties.
      </p>

      <h3>How We Use Your Information</h3>
      <ul>
        <li>Deliver, maintain, and improve the StatChat platform.</li>
        <li>Authenticate users via the StatGate Identity service.</li>
        <li>Provide real-time messaging, notifications, and collaboration features.</li>
        <li>Facilitate calendar scheduling and meeting room booking.</li>
        <li>Enforce organisational security policies, including Attribute-Based Access Control (ABAC).</li>
        <li>Generate anonymised aggregate analytics for enterprise administrators.</li>
        <li>Comply with legal obligations and respond to lawful requests.</li>
      </ul>

      <h3>Data Sharing & Disclosure</h3>
      <p>
        StatGate does not sell, rent, or trade your personal information. We may share data only in the
        following limited circumstances:
      </p>
      <ul>
        <li>
          <strong>Within your organisation:</strong> Messages and files are visible to members of the
          conversations, teams, and channels you participate in, as governed by your organisation&rsquo;s ABAC
          policies.
        </li>
        <li>
          <strong>Service Providers:</strong> Trusted third-party sub-processors (cloud infrastructure,
          email delivery) who are contractually bound to equivalent data-protection standards.
        </li>
        <li>
          <strong>Legal Compliance:</strong> When required by applicable law, regulation, or valid legal
          process, and only after notifying your organisation&rsquo;s designated data controller unless
          prohibited by law.
        </li>
      </ul>

      <h3>Data Retention</h3>
      <p>
        Your organisation&rsquo;s administrator defines data retention policies. By default, StatChat retains
        messages and files for the duration of the organisation&rsquo;s active subscription, unless an earlier
        deletion is requested. Deleted content is irrecoverably purged from our systems within 30 days.
      </p>

      <h3>Your Rights</h3>
      <p>
        Depending on your jurisdiction, you may have the right to access, rectify, erase, restrict, or port
        your personal data. Requests should be directed to your organisation&rsquo;s StatGate administrator in
        the first instance. StatGate will assist administrators in fulfilling data-subject requests within the
        timeframes mandated by applicable law.
      </p>

      <h3>Security</h3>
      <p>
        StatChat employs industry-standard encryption (TLS 1.3 in transit, AES-256 at rest), multi-factor
        authentication, role-based and attribute-based access controls, continuous monitoring, and regular
        third-party penetration testing. If you suspect a security vulnerability, please report it to our
        security team immediately.
      </p>

      <h3>Contact</h3>
      <p>
        For privacy-related inquiries, contact <strong>privacy@statchat.local</strong> or your
        organisation&rsquo;s Data Protection Officer. Our Data Protection Officer can be reached at{' '}
        <strong>dpo@statchat.local</strong>.
      </p>

      <p className={styles.lastUpdated}>
        Last updated: 1 August 2026 &middot; Version 2.4
      </p>
    </>
  );
}

function TermsContent() {
  return (
    <>
      <h2>Terms of Service</h2>
      <p>
        Welcome to StatChat. These Terms of Service (&ldquo;Terms&rdquo;) govern your access to and use of
        the StatChat enterprise communication and collaboration platform, including all associated services,
        websites, and software (collectively, &ldquo;the Service&rdquo;). By accessing or using StatChat, you
        agree to be bound by these Terms.
      </p>

      <h3>1. Eligibility & Account</h3>
      <p>
        You must be an authorised member of a StatGate-subscribed organisation to use StatChat. You are
        responsible for maintaining the confidentiality of your login credentials and for all activities
        that occur under your account. Notify your organisation&rsquo;s administrator immediately of any
        unauthorised use.
      </p>

      <h3>2. Acceptable Use</h3>
      <p>You agree not to:</p>
      <ul>
        <li>Use the Service for any unlawful purpose or in violation of any applicable laws or regulations.</li>
        <li>Upload, transmit, or share content that is defamatory, harassing, hateful, or infringes the
        intellectual property rights of any third party.</li>
        <li>Attempt to interfere with, compromise, or gain unauthorised access to the Service, its servers,
        or the data of other users.</li>
        <li>Use automated means (bots, scrapers) to access or extract data from the Service without express
        written permission.</li>
        <li>Resell, sublicense, or commercially exploit the Service outside the scope of your
        organisation&rsquo;s licence agreement.</li>
      </ul>

      <h3>3. Intellectual Property</h3>
      <p>
        StatGate retains all right, title, and interest in and to the Service, including all underlying
        software, algorithms, designs, and documentation. Content you create and share through StatChat
        remains your own. By submitting content, you grant StatGate and your organisation a limited,
        non-exclusive licence to process, store, and display that content solely for the purpose of
        operating the Service.
      </p>

      <h3>4. Third-Party Integrations</h3>
      <p>
        StatChat may integrate with third-party services (e.g., calendar providers, video conferencing
        platforms, cloud storage). Your use of such integrations is subject to the terms and privacy
        policies of the respective third-party providers. StatGate disclaims all liability for the acts
        or omissions of third-party services.
      </p>

      <h3>5. Service Availability</h3>
      <p>
        StatGate strives to maintain a Service Level Objective (SLO) of 99.9% monthly uptime for the
        StatChat platform. Scheduled maintenance is announced at least 48 hours in advance via the
        System Status page. StatGate shall not be liable for interruptions caused by factors beyond its
        reasonable control, including force majeure events, internet infrastructure failures, or actions
        of third-party hosting providers.
      </p>

      <h3>6. Limitation of Liability</h3>
      <p>
        To the fullest extent permitted by applicable law, StatGate&rsquo;s aggregate liability for all claims
        arising out of or relating to these Terms or the Service shall not exceed the fees paid by your
        organisation for the Service during the twelve (12) months preceding the event giving rise to the
        claim. In no event shall StatGate be liable for indirect, incidental, special, or consequential
        damages, including lost profits or data loss.
      </p>

      <h3>7. Termination</h3>
      <p>
        Your organisation may terminate your access at any time. StatGate reserves the right to suspend or
        terminate accounts that violate these Terms. Upon termination, your right to access StatChat ceases
        immediately. Your organisation&rsquo;s data retention policy governs the disposition of your content
        post-termination.
      </p>

      <h3>8. Changes to These Terms</h3>
      <p>
        StatGate may revise these Terms from time to time. Material changes will be communicated to your
        organisation&rsquo;s administrator at least thirty (30) days before they take effect. Continued use
        of the Service after the effective date constitutes acceptance of the revised Terms.
      </p>

      <h3>9. Governing Law & Disputes</h3>
      <p>
        These Terms are governed by the laws applicable to your organisation&rsquo;s principal place of
        business, as specified in the Master Services Agreement between StatGate and your organisation.
        Any disputes shall be resolved through binding arbitration in accordance with the rules of the
        relevant arbitration body, unless otherwise agreed in writing.
      </p>

      <p className={styles.lastUpdated}>
        Last updated: 15 June 2026 &middot; Version 3.1
      </p>
    </>
  );
}

function StatusContent() {
  const lowPct = 'less than 2%';

  return (
    <>
      <h2>System Status</h2>

      <div className={styles.statusIndicator}>
        <span className={styles.statusDot} />
        <span className={styles.statusText}>
          All systems operational
          <small>Last refreshed: just now</small>
        </span>
      </div>

      <h3>Service Uptime (Last 90 Days)</h3>
      <div className={styles.statusServices}>
        {serviceStatuses.map((svc) => (
          <div key={svc.name} className={styles.serviceRow}>
            <div>
              <div className={styles.serviceName}>{svc.name}</div>
              <div className={styles.serviceUptime}>{svc.uptime} uptime</div>
            </div>
            <span
              className={`${styles.serviceStatus} ${
                svc.operational ? styles.serviceStatusOperational : styles.serviceStatusDown
              }`}
            >
              <span
                style={{
                  width: 8,
                  height: 8,
                  borderRadius: '50%',
                  background: svc.operational ? '#16a34a' : '#dc2626',
                  display: 'inline-block',
                }}
              />
              {svc.operational ? 'Operational' : 'Degraded'}
            </span>
          </div>
        ))}
      </div>

      <h3>Incident History</h3>
      <p>
        <strong>1 August 2026:</strong> Scheduled maintenance of the WebSocket Gateway completed successfully.
        No user impact observed. Duration: 12 minutes during off-peak hours (02:00&ndash;02:12 UTC).
      </p>
      <p>
        <strong>28 July 2026:</strong> Brief latency spike on the Search Indexer due to an upstream
        elasticsearch rebalancing operation. Full resolution within 8 minutes. Affected {lowPct} of queries.
      </p>
      <p>
        <strong>12 July 2026:</strong> Push notification delivery delay for approximately 4% of Android
        devices caused by an FCM regional routing issue. Resolved by Google at 14:32 UTC. StatChat
        notifications queued and delivered within the retry window.
      </p>

      <h3>Uptime Commitment</h3>
      <p>
        StatGate maintains a Service Level Objective (SLO) of <strong>99.9%</strong> monthly uptime for
        StatChat core messaging services. Current trailing-twelve-month uptime: <strong>99.97%</strong>.
        For enterprise SLA details, please refer to your Master Services Agreement or contact your
        StatGate account representative.
      </p>

      <p className={styles.lastUpdated}>
        Real-time status updates are also available at <strong>status.statchat.local</strong>
      </p>
    </>
  );
}

export default function LegalPanel({ open, initialTab = 'privacy', onClose }: Props) {
  const [activeTab, setActiveTab] = useState<LegalTab>(initialTab);

  if (!open) return null;

  const content = (() => {
    switch (activeTab) {
      case 'privacy':
        return <PrivacyContent />;
      case 'terms':
        return <TermsContent />;
      case 'status':
        return <StatusContent />;
    }
  })();

  return (
    <div className={styles.overlay} onClick={onClose}>
      <div className={styles.panel} onClick={(e) => e.stopPropagation()}>
        <div className={styles.header}>
          <div className={styles.headerBrand}>
            <span className={styles.headerLogo}>🚀</span>
            <span className={styles.headerBrandName}>StatChat</span>
          </div>
          <button
            type="button"
            className={styles.backButton}
            onClick={onClose}
            aria-label="Close panel"
          >
            ✕
          </button>
        </div>

        <div className={styles.tabBar}>
          {tabs.map((tab) => (
            <button
              key={tab.id}
              type="button"
              className={`${styles.tab} ${activeTab === tab.id ? styles.tabActive : ''}`}
              onClick={() => setActiveTab(tab.id)}
            >
              {tab.label}
            </button>
          ))}
        </div>

        <div className={styles.body}>
          {content}
        </div>
      </div>
    </div>
  );
}