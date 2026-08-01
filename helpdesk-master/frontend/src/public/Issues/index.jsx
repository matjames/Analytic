
import React, { useState, Fragment } from "react";
import { useHistory } from "react-router-dom";
import { toast } from "react-toastify";
import Select from "react-select";
import API from "../../helpers/api";
import { Spinner } from "react-bootstrap";
import healthFacilities from "../../data/facilities";

const Issues = () => {
  const [facility, setFacility] = useState("");
  const [loading, setLoading] = useState(false);
  const [reportedby, setReportedBy] = useState("");
  const [system, setSystem] = useState("Analytics");
  const [emrtype, setEMRType] = useState("");
  const [module, setModule] = useState("");
  const [phoneno, setPhoneNo] = useState("");
  const [description, setDescription] = useState("");
  const [priority, setPriority] = useState("");
  const [level, setLevel] = useState("");
  const [category, setCategory] = useState("");
  const [file, setFile] = useState("");
  const [dhis2instance, setDHIS2Instance] = useState("");
  const [dhis2module, setDHIS2Module] = useState("");

  const history = useHistory();

  const handleChange = (event) => setCategory(event.target.value);
  const handleSystemChange = (event) => setSystem(event.target.value);
  const handleSelectChange = (selectedOption) => setFacility(selectedOption);
  const handleFileChange = (event) => setFile(event.target.files[0]);

  const filteredHealthFacilities = healthFacilities.filter(
    (facility) => facility.level === level
  );

  const handleSubmit = async (e) => {
    e.preventDefault();
    setLoading(true);

    const formData = new FormData();

    formData.append('reportedby', reportedby);
    formData.append('priority', priority);
    formData.append('level', level);
    formData.append('facility', facility.value);
    formData.append('category', category);
    formData.append('module', module);
    formData.append('emrtype', emrtype);
    formData.append('phoneno', phoneno);
    formData.append('description', description);
    formData.append('agentId', 1);
    formData.append('image', file);
    formData.append('system', system);
    formData.append('dhis2instance', dhis2instance);
    formData.append('dhis2module', dhis2module);

    try {
      await API.post('/t/tickets', formData, {
        headers: {
          'Content-Type': 'multipart/form-data'
        }
      });
      setLoading(false);
      history.push('/public/tickets');
      toast.success('Work item submitted successfully');
    } catch (err) {
      setLoading(false);
    }
  };

  return (
    <Fragment>
      <div class="content-area">
        <div class="form-container">
          <div class="form-header">
            <div class="platform-logo">
              <div class="logo-icon">
                <i class="fas fa-chart-line"></i>
              </div>
              <h1 class="platform-title">StatGate Operations Center</h1>
            </div>
            <p class="text-muted">Log an analytics incident, data quality issue, workflow blocker, or access request.</p>
          </div>

          <form onSubmit={handleSubmit}>
            <div class="row">
              <div class="col-md-6 mb-3">
                <label htmlFor="reportedBy" class="form-label">Reported By</label>
                <input type="text" class="form-control" id="reportedBy" placeholder="Enter reporter name" value={reportedby} onChange={(e) => setReportedBy(e.target.value)} />
              </div>
              <div class="col-md-6 mb-3">
                <label htmlFor="phoneNumber" class="form-label">Phone Number</label>
                <input type="tel" class="form-control" id="phoneNumber" placeholder="Enter phone number" value={phoneno} onChange={(e) => setPhoneNo(e.target.value)} />
              </div>
            </div>

            <div class="row">
              <div class="col-md-6 mb-3">
                <label htmlFor="facilityLevel" class="form-label">Program / Unit</label>
                <select class="form-select" id="facilityLevel" value={level} onChange={(e) => setLevel(e.target.value)}>
                  <option value="">Select program level</option>
                  <option value="National">National</option>
                  <option value="Regional">Regional</option>
                  <option value="District">District</option>
                  <option value="Facility">Facility</option>
                </select>
              </div>
              <div class="col-md-6 mb-3">
                <label htmlFor="healthFacility" class="form-label">Service Area</label>
                <Select options={filteredHealthFacilities} value={facility} onChange={handleSelectChange} placeholder="Select service area" isSearchable />
              </div>
            </div>

            <div class="row">
              <div class="col-md-4 mb-3">
                <label htmlFor="systemCategory" class="form-label">Workstream</label>
                <select class="form-select" id="systemCategory" value={system} onChange={handleSystemChange}>
                  <option value="Analytics">Analytics</option>
                  <option value="Data Pipeline">Data Pipeline</option>
                  <option value="Governance">Governance</option>
                  <option value="Reporting">Reporting</option>
                  <option value="Operations">Operations</option>
                </select>
              </div>
              <div class="col-md-4 mb-3">
                <label htmlFor="emrModule" class="form-label">Operational Module</label>
                <select class="form-select" id="emrModule" value={module} onChange={(e) => setModule(e.target.value)}>
                  <option value="">Select module</option>
                  <option value="Dashboard">Dashboard</option>
                  <option value="Data Quality">Data Quality</option>
                  <option value="Workflow">Workflow</option>
                  <option value="Access Management">Access Management</option>
                  <option value="Alerting">Alerting</option>
                </select>
              </div>
              <div class="col-md-4 mb-3">
                <label htmlFor="priority" class="form-label">Severity</label>
                <select class="form-select" id="priority" value={priority} onChange={(e) => setPriority(e.target.value)}>
                  <option value="">Select severity</option>
                  <option value="Low">Low</option>
                  <option value="Medium">Medium</option>
                  <option value="High">High</option>
                  <option value="Critical">Critical</option>
                </select>
              </div>
            </div>

            <div class="mb-4">
              <label htmlFor="issueDescription" class="form-label">What happened? Share the impact and expected outcome</label>
              <textarea class="form-control" id="issueDescription" rows="4" placeholder="Describe the incident, observed behavior, and urgency" value={description} onChange={(e) => setDescription(e.target.value)}></textarea>
            </div>

            <div class="mb-4">
              <label class="form-label">Attach evidence</label>
              <div class="file-upload-area">
                <div class="file-upload-icon">
                  <i class="fas fa-cloud-upload-alt"></i>
                </div>
                <h6>Choose File or Drag & Drop</h6>
                <p class="text-muted small">Upload screenshots, exports, or supporting documents</p>
                <input type="file" class="form-control" id="fileUpload" multiple accept="image/*,.pdf,.doc,.docx" style={{ display: 'none' }} onChange={handleFileChange} />
                <button type="button" class="btn btn-outline-primary btn-sm" onClick={() => document.getElementById('fileUpload').click()}>
                  <i class="fas fa-folder-open me-2"></i>Browse Files
                </button>
              </div>
            </div>

            <div class="text-center">
              <button type="submit" class="btn btn-primary" disabled={loading}>
                {loading ? <Spinner animation="border" size="sm" /> : <><i class="fas fa-paper-plane me-2"></i>Submit Work Item</>}
              </button>
            </div>
          </form>
        </div>
      </div>

      <div class="chat-widget">
        <button class="chat-btn">
          <i class="fas fa-comment me-2"></i>
          Operations Chat
        </button>
      </div>
    </Fragment>
  );
};

export default Issues;