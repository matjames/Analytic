import React, { useState } from 'react'
import './styles.css';

const FAQ = () => {
  const [activeCategory, setActiveCategory] = useState('all');
  const [openQuestions, setOpenQuestions] = useState(new Set());
  const [searchTerm, setSearchTerm] = useState('');

  const handleCategoryChange = (category) => {
    setActiveCategory(category);
  };

  const toggleQuestion = (questionId) => {
    const newOpenQuestions = new Set(openQuestions);
    if (newOpenQuestions.has(questionId)) {
      newOpenQuestions.delete(questionId);
    } else {
      newOpenQuestions.add(questionId);
    }
    setOpenQuestions(newOpenQuestions);
  };

  const handleSearch = (e) => {
    setSearchTerm(e.target.value);
  };

  const categories = [
    { id: 'all', label: 'All' },
    { id: 'workitems', label: 'Work Items' },
    { id: 'account', label: 'Access' },
    { id: 'technical', label: 'Technical' },
    { id: 'integrations', label: 'Integrations' }
  ];

  const faqData = [
    {
      id: 'ticket-1',
      category: 'workitems',
      question: 'How do I create a new work item?',
      answer: 'Open the Operations Work Items page, choose the work type and severity, then submit the details and any evidence needed for triage.'
    },
    {
      id: 'ticket-2',
      category: 'workitems',
      question: 'How can I update an existing work item?',
      answer: 'Open the item from the work queue, add a status update, and share any new information so the operations team can respond quickly.'
    },
    {
      id: 'ticket-3',
      category: 'workitems',
      question: 'What are the different work item statuses?',
      answer: 'Work items can be New, In Progress, Pending, Resolved, or Closed depending on the current stage of the response.'
    },
    {
      id: 'account-1',
      category: 'account',
      question: 'How do I reset my password?',
      answer: 'Click on the "Forgot Password" link on the login page, enter your email address, and follow the instructions sent to your email to reset your password.'
    },
    {
      id: 'account-2',
      category: 'account',
      question: 'How can I update my profile information?',
      answer: 'Go to your user profile by clicking on your name in the top right corner, then select "Profile Settings" to update your personal information, contact details, and preferences.'
    },
    {
      id: 'technical-1',
      category: 'technical',
      question: 'What browsers are supported?',
      answer: 'Our platform supports the latest versions of Chrome, Firefox, Safari, and Edge. For the best experience, we recommend using Chrome or Firefox with JavaScript enabled.'
    },
    {
      id: 'technical-2',
      category: 'technical',
      question: 'Why am I experiencing slow loading times?',
      answer: 'Slow loading can be caused by internet connectivity, browser cache, or high server load. Try clearing your browser cache, checking your internet connection, or contacting support if the issue persists.'
    },
    {
      id: 'integrations-1',
      category: 'integrations',
      question: 'How do I connect email integration?',
      answer: 'Go to Settings Integrations, select Email Integration, and follow the setup wizard to connect your email provider. You\'ll need to provide your email server settings or OAuth credentials.'
    },
    {
      id: 'integrations-2',
      category: 'integrations',
      question: 'Can I integrate with third-party tools?',
      answer: 'Yes, we support integrations with popular tools like Slack, Microsoft Teams, Jira, and many others. Check our integrations marketplace for available options and setup instructions.'
    }
  ];

  const filteredFaqData = faqData.filter(item => {
    const matchesCategory = activeCategory === 'all' || item.category === activeCategory;
    const matchesSearch = searchTerm === '' || 
      item.question.toLowerCase().includes(searchTerm.toLowerCase()) ||
      item.answer.toLowerCase().includes(searchTerm.toLowerCase());
    return matchesCategory && matchesSearch;
  });

  const groupedFaqData = filteredFaqData.reduce((acc, item) => {
    if (!acc[item.category]) {
      acc[item.category] = [];
    }
    acc[item.category].push(item);
    return acc;
  }, {});

  return (
    <>
      <div className="content-area">
        <nav className="breadcrumb-nav">
          <ol className="breadcrumb">
            <li className="breadcrumb-item"><a href="index.html">Operations</a></li>
            <li className="breadcrumb-item active">FAQ</li>
          </ol>
        </nav>

        <div className="page-header">
          <h1 className="page-title">
            <i className="fas fa-question-circle me-2"></i>
            Frequently Asked Questions
          </h1>
          <p className="page-subtitle">Find answers to common questions about the StatGate operations experience</p>
        </div>

        <div className="search-section">
          <div className="search-bar">
            <input 
              type="text" 
              className="search-input" 
              placeholder="Search for answers..." 
              value={searchTerm}
              onChange={handleSearch}
            />
            <i className="fas fa-search search-icon"></i>
          </div>
          <div className="filter-tabs">
            {categories.map(category => (
              <a 
                key={category.id}
                href="#" 
                className={`filter-tab ${activeCategory === category.id ? 'active' : ''}`}
                onClick={(e) => {
                  e.preventDefault();
                  handleCategoryChange(category.id);
                }}
              >
                {category.label}
              </a>
            ))}
          </div>
        </div>

        <div className="faq-container">

          {Object.keys(groupedFaqData).length > 0 ? (
            Object.entries(groupedFaqData).map(([category, items]) => (
              <div key={category} className="faq-section">
                <h2 className="section-title">
                  {category === 'workitems' && 'Work Item Management'}
                  {category === 'account' && 'Account & Access'}
                  {category === 'technical' && 'Technical Support'}
                  {category === 'integrations' && 'Integrations'}
                </h2>
                {items.map(item => (
                  <div key={item.id} className="faq-item">
                    <button 
                      className={`faq-question ${openQuestions.has(item.id) ? 'active' : ''}`}
                      onClick={() => toggleQuestion(item.id)}
                    >
                      {item.question}
                      <i className="fas fa-chevron-down faq-icon"></i>
                    </button>
                    <div className={`faq-answer ${openQuestions.has(item.id) ? 'active' : ''}`}>
                      <div className="faq-answer-content">
                        {item.answer}
                      </div>
                    </div>
                  </div>
                ))}
              </div>
            ))
          ) : (
            <div className="faq-section">
              <div className="no-results">
                <h3>No results found</h3>
                <p>Try adjusting your search terms or browse all categories.</p>
              </div>
            </div>
          )}

          <div className="contact-support">
            <h3 className="contact-title">Still need help?</h3>
            <p className="contact-text">Can't find the answer you're looking for? The operations team can help you get to the right workflow.</p>
            <a href="add-ticket.html" className="contact-btn">Contact Operations</a>
          </div>
        </div>
      </div>
    </>

  )
}

export default FAQ