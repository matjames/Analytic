import React, { useState, useEffect } from 'react';
import { Plus, Trash2, Download, QrCode, CheckCircle } from 'lucide-react';

interface Question {
  id: string;
  code: string;
  questionText: string;
  responseType: 'single_choice' | 'multiple_choice' | 'numeric' | 'text' | 'date' | 'geo_point';
  options: string[];
  required: boolean;
  domain: string;
}

export const FormDesigner: React.FC = () => {
  const [formTitle, setFormTitle] = useState('National Household Welfare Survey 2026');
  const [formId, setFormId] = useState('HH_WELFARE_2026');
  const [questions, setQuestions] = useState<Question[]>([
    {
      id: 'q1',
      code: 'Q_DEMO_AGE_01',
      questionText: 'How old was the respondent on their last birthday?',
      responseType: 'numeric',
      options: [],
      required: true,
      domain: 'Demographics'
    },
    {
      id: 'q2',
      code: 'Q_HLTH_ACCESS_01',
      questionText: 'In the past 30 days, did any household member visit a formal health facility?',
      responseType: 'single_choice',
      options: ['Yes', 'No', "Don't know"],
      required: true,
      domain: 'Health'
    }
  ]);

  const [questionBank, setQuestionBank] = useState<any[]>([]);
  const [showBankPicker, setShowBankPicker] = useState(false);
  const [showQrModal, setShowQrModal] = useState(false);
  const [published, setPublished] = useState(false); // read below as status badge

  useEffect(() => {
    fetch('http://localhost:8096/api/statistics/question-bank')
      .then(res => res.json())
      .then(data => setQuestionBank(Array.isArray(data) ? data : []))
      .catch(() => {});
  }, []);

  const addQuestion = (type: Question['responseType'] = 'text') => {
    const newQ: Question = {
      id: `q_${Date.now()}`,
      code: `Q_${questions.length + 1}`,
      questionText: 'New survey question',
      responseType: type,
      options: type === 'single_choice' || type === 'multiple_choice' ? ['Option 1', 'Option 2'] : [],
      required: true,
      domain: 'General'
    };
    setQuestions([...questions, newQ]);
  };

  const removeQuestion = (id: string) => {
    setQuestions(questions.filter(q => q.id !== id));
  };

  const handleImportFromBank = (item: any) => {
    const newQ: Question = {
      id: `q_${Date.now()}`,
      code: item.code || `Q_${questions.length + 1}`,
      questionText: item.question_text || item.QuestionText,
      responseType: item.response_type || 'text',
      options: item.options || [],
      required: true,
      domain: item.domain || 'Demographics'
    };
    setQuestions([...questions, newQ]);
    setShowBankPicker(false);
  };

  const handleExportXForm = () => {
    const xml = `<?xml version="1.0" encoding="utf-8"?>
<h:html xmlns="http://www.w3.org/2002/xforms" xmlns:h="http://www.w3.org/1999/xhtml" xmlns:jr="http://openrosa.org/javarosa">
  <h:head>
    <h:title>${formTitle}</h:title>
    <model>
      <instance>
        <data id="${formId}">
          ${questions.map(q => `<${q.code}/>`).join('\n          ')}
        </data>
      </instance>
      ${questions.map(q => `<bind nodeset="/data/${q.code}" type="${q.responseType === 'numeric' ? 'int' : 'string'}" ${q.required ? 'required="true()"' : ''}/>`).join('\n      ')}
    </model>
  </h:head>
  <h:body>
    ${questions.map(q => `<input ref="/data/${q.code}"><label>${q.questionText}</label></input>`).join('\n    ')}
  </h:body>
</h:html>`;

    const blob = new Blob([xml], { type: 'application/xml' });
    const url = URL.createObjectURL(blob);
    const a = document.createElement('a');
    a.href = url;
    a.download = `${formId}.xml`;
    a.click();
  };

  return (
    <div className="bg-white rounded-xl shadow-sm border border-gray-200 p-6">
      {/* Header */}
      <div className="flex justify-between items-center pb-6 border-b border-gray-200">
        <div>
          <div className="flex items-center gap-3">
            <span className="text-2xl">📝</span>
            <input
              type="text"
              value={formTitle}
              onChange={e => setFormTitle(e.target.value)}
              className="text-xl font-bold text-gray-900 border-b border-dashed border-gray-400 focus:border-blue-600 outline-none pb-0.5"
            />
          </div>
          <div className="flex items-center gap-2 mt-1 text-xs text-gray-500">
            <span>Form ID:</span>
            <input
              type="text"
              value={formId}
              onChange={e => setFormId(e.target.value.toUpperCase().replace(/\s/g, '_'))}
              className="font-mono bg-gray-100 px-1.5 py-0.5 rounded border border-gray-200 focus:border-blue-400 outline-none text-gray-700 w-44"
            />
            <span>| GSBPM 3.1 · OpenRosa / ODK XForms</span>
          </div>
        </div>

        <div className="flex items-center gap-2">
          <button
            onClick={() => setShowBankPicker(true)}
            className="flex items-center gap-1.5 px-3 py-2 text-xs font-semibold bg-gray-100 hover:bg-gray-200 text-gray-700 rounded-lg transition"
          >
            📚 Question Bank
          </button>
          <button
            onClick={handleExportXForm}
            className="flex items-center gap-1.5 px-3 py-2 text-xs font-semibold bg-blue-50 hover:bg-blue-100 text-blue-700 rounded-lg transition"
          >
            <Download className="w-4 h-4" /> Export XForm (XML)
          </button>
          {published && (
            <span className="flex items-center gap-1 text-[10px] px-2 py-1 bg-green-100 text-green-700 rounded-full font-semibold border border-green-200">
              <CheckCircle className="w-3 h-3" /> Published
            </span>
          )}
          <button
            onClick={() => { setPublished(true); setShowQrModal(true); }}
            className="flex items-center gap-1.5 px-3.5 py-2 text-xs font-semibold bg-blue-600 hover:bg-blue-700 text-white rounded-lg transition shadow-sm"
          >
            <QrCode className="w-4 h-4" /> Publish &amp; QR Deploy
          </button>
        </div>
      </div>

      {/* Question List */}
      <div className="py-6 space-y-4">
        {questions.map((q, idx) => (
          <div key={q.id} className="p-4 bg-gray-50 border border-gray-200 rounded-lg hover:border-blue-300 transition">
            <div className="flex justify-between items-start mb-3">
              <div className="flex items-center gap-2">
                <span className="w-6 h-6 rounded-full bg-blue-100 text-blue-800 text-xs font-bold flex items-center justify-center">
                  {idx + 1}
                </span>
                <input
                  type="text"
                  value={q.code}
                  onChange={e => {
                    const updated = [...questions];
                    updated[idx].code = e.target.value;
                    setQuestions(updated);
                  }}
                  className="text-xs font-mono bg-white px-2 py-1 rounded border border-gray-300 text-gray-700 w-36"
                />
                <span className="text-xs px-2 py-0.5 bg-gray-200 text-gray-600 rounded">
                  {q.domain}
                </span>
              </div>
              <button
                onClick={() => removeQuestion(q.id)}
                className="text-gray-400 hover:text-red-600 p-1 transition"
              >
                <Trash2 className="w-4 h-4" />
              </button>
            </div>

            <input
              type="text"
              value={q.questionText}
              onChange={e => {
                const updated = [...questions];
                updated[idx].questionText = e.target.value;
                setQuestions(updated);
              }}
              className="w-full text-sm font-medium text-gray-900 bg-white p-2.5 rounded border border-gray-300 focus:border-blue-500 outline-none mb-3"
            />

            <div className="flex items-center gap-4 text-xs text-gray-600">
              <div className="flex items-center gap-1.5">
                <span className="font-semibold">Type:</span>
                <select
                  value={q.responseType}
                  onChange={e => {
                    const updated = [...questions];
                    updated[idx].responseType = e.target.value as any;
                    setQuestions(updated);
                  }}
                  className="bg-white border border-gray-300 rounded px-2 py-1"
                >
                  <option value="text">Text (String)</option>
                  <option value="numeric">Numeric (Integer/Decimal)</option>
                  <option value="single_choice">Single Choice (Radio)</option>
                  <option value="multiple_choice">Multiple Choice (Checkbox)</option>
                  <option value="date">Date / Timestamp</option>
                  <option value="geo_point">GPS GeoPoint</option>
                </select>
              </div>

              <label className="flex items-center gap-1.5 cursor-pointer">
                <input
                  type="checkbox"
                  checked={q.required}
                  onChange={e => {
                    const updated = [...questions];
                    updated[idx].required = e.target.checked;
                    setQuestions(updated);
                  }}
                  className="rounded text-blue-600"
                />
                <span>Mandatory field</span>
              </label>
            </div>
          </div>
        ))}
      </div>

      {/* Add Button */}
      <div className="flex gap-2 pt-2 border-t border-gray-200">
        <button
          onClick={() => addQuestion('text')}
          className="flex items-center gap-1.5 px-3 py-2 text-xs font-semibold bg-gray-100 hover:bg-gray-200 text-gray-800 rounded-lg transition"
        >
          <Plus className="w-4 h-4" /> Add Text Question
        </button>
        <button
          onClick={() => addQuestion('numeric')}
          className="flex items-center gap-1.5 px-3 py-2 text-xs font-semibold bg-gray-100 hover:bg-gray-200 text-gray-800 rounded-lg transition"
        >
          <Plus className="w-4 h-4" /> Add Numeric Question
        </button>
        <button
          onClick={() => addQuestion('single_choice')}
          className="flex items-center gap-1.5 px-3 py-2 text-xs font-semibold bg-gray-100 hover:bg-gray-200 text-gray-800 rounded-lg transition"
        >
          <Plus className="w-4 h-4" /> Add Choice Question
        </button>
      </div>

      {/* Question Bank Picker Modal */}
      {showBankPicker && (
        <div className="fixed inset-0 bg-black bg-opacity-50 flex items-center justify-center z-50 p-4">
          <div className="bg-white rounded-xl max-w-2xl w-full p-6 shadow-xl max-h-[80vh] flex flex-col">
            <h3 className="text-lg font-bold text-gray-900 mb-2">Standardized Question Bank (DDI/SDMX)</h3>
            <p className="text-xs text-gray-500 mb-4">Select pre-harmonized questions to maintain longitudinal and cross-survey statistical comparability.</p>
            <div className="overflow-y-auto flex-1 space-y-2.5 pr-2">
              {questionBank.map(item => (
                <div
                  key={item.id}
                  onClick={() => handleImportFromBank(item)}
                  className="p-3 bg-gray-50 hover:bg-blue-50 border border-gray-200 hover:border-blue-300 rounded-lg cursor-pointer transition"
                >
                  <div className="flex justify-between items-center mb-1">
                    <span className="font-mono text-xs text-blue-700 font-bold">{item.code}</span>
                    <span className="text-xs px-2 py-0.5 bg-gray-200 text-gray-700 rounded font-medium">{item.domain}</span>
                  </div>
                  <div className="text-xs text-gray-800">{item.question_text || item.QuestionText}</div>
                </div>
              ))}
            </div>
            <div className="mt-4 flex justify-end">
              <button onClick={() => setShowBankPicker(false)} className="px-4 py-2 bg-gray-200 hover:bg-gray-300 text-xs font-semibold rounded-lg">
                Close
              </button>
            </div>
          </div>
        </div>
      )}

      {/* QR Deploy Modal */}
      {showQrModal && (
        <div className="fixed inset-0 bg-black bg-opacity-50 flex items-center justify-center z-50 p-4">
          <div className="bg-white rounded-xl max-w-md w-full p-6 text-center shadow-xl">
            <div className="w-12 h-12 rounded-full bg-green-100 text-green-600 flex items-center justify-center mx-auto mb-3">
              <CheckCircle className="w-6 h-6" />
            </div>
            <h3 className="text-lg font-bold text-gray-900 mb-1">Form Published to StatCollect</h3>
            <p className="text-xs text-gray-600 mb-4">Scan QR code in ODK Collect or collect-master (Android) to download form immediately.</p>
            
            <div className="bg-gray-100 p-6 rounded-lg flex items-center justify-center mb-4 border border-dashed border-gray-300">
              <div className="w-40 h-40 bg-white p-2 rounded shadow flex items-center justify-center">
                <QrCode className="w-32 h-32 text-gray-900" />
              </div>
            </div>

            <div className="text-xs text-gray-500 font-mono bg-gray-50 p-2 rounded mb-4">
              Server: http://localhost:8080/formList
            </div>

            <button
              onClick={() => setShowQrModal(false)}
              className="w-full py-2 bg-blue-600 hover:bg-blue-700 text-white text-xs font-semibold rounded-lg"
            >
              Done
            </button>
          </div>
        </div>
      )}
    </div>
  );
};
