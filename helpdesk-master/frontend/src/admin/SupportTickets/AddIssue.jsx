
import React, { useState, Fragment } from "react";
import { useHistory } from "react-router-dom";
import { toast } from "react-toastify";
import Select from "react-select";
import API from "../../helpers/api";
import { Spinner } from "react-bootstrap";
import healthFacilities from "../../data/facilities";

const AddIssue = () => {
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
    formData.append('agentId', '6dc80317-76cb-4fcd-968a-f53b32e56734');
    formData.append('image', file);
    formData.append('system', system);
    formData.append('dhis2instance', dhis2instance);
    formData.append('dhis2module', dhis2module);

    try {
      const response = await API.post('/t/tickets', formData, {
        headers: {
          'Content-Type': 'multipart/form-data'
        }
      });
      setLoading(false);
      history.push('/public/tickets');
      toast.success('Work item created successfully');
    } catch (err) {
      setLoading(false);
    }
  };

  return (
    <Fragment>
      <div className="row justify-content-center">
        <div className="col-10">
          <div className="card overflow-hidden">
            <div className="card-body pt-0">
              <div className="p-4">
                <div className="row">
                  <div className="col-6">
                    <div className="mb-3">
                      <label htmlFor="username" className="form-label">
                        Reported By
                      </label>
                      <input
                        type="text"
                        className="form-control"
                        placeholder="Enter reporter name"
                        value={reportedby}
                        onChange={(e) => setReportedBy(e.target.value)}
                      />
                    </div>
                  </div>
                  <div className="col-6">
                    <div className="mb-3">
                      <label htmlFor="useremail" className="form-label">
                        Phone Number
                      </label>
                      <input
                        type="text"
                        className="form-control"
                        placeholder="Enter phone number"
                        value={phoneno}
                        onChange={(e) => setPhoneNo(e.target.value)}
                      />
                    </div>
                  </div>
                </div>

                <div className="row">
                  <div className="col-6">
                    <div className="mb-3">
                      <label htmlFor="useremail" className="form-label">
                        Program / Service Unit
                      </label>
                      <select
                        className="form-select"
                        aria-label="Select example"
                        value={level}
                        onChange={(e) => setLevel(e.target.value)}
                      >
                        <option value="">Select Program Level</option>
                        <option value="National Referral">
                          National Referral
                        </option>
                        <option value="Regional Referral">
                          Regional Referral
                        </option>
                        <option value="General Hospital">
                          General Hospital
                        </option>
                        <option value="HC IV">HC IV</option>
                        <option value="HC III">HC III</option>
                      </select>
                    </div>
                  </div>
                  <div className="col-6">
                    <div className="mb-3">
                      <label htmlFor="username" className="form-label">
                        Operational Area
                      </label>
                      <Select
                        options={filteredHealthFacilities}
                        value={facility}
                        onChange={handleSelectChange}
                        placeholder="Select operational area"
                        isSearchable
                      />
                    </div>
                  </div>
                </div>
                <div className="row">
                  <div className="col-6">
                    <div className="mb-3">
                      <label htmlFor="useremail" className="form-label">
                        Workstream
                      </label>
                      <select
                        className="form-select"
                        aria-label="Select example"
                        value={system}
                        onChange={handleSystemChange}
                      >
                        <option value="">Select System Category</option>
                        <option value="Analytics">Analytics</option>
                        <option value="Data Pipeline">Data Pipeline</option>
                        <option value="Governance">Governance</option>
                        <option value="Reporting">Reporting</option>
                      </select>
                    </div>
                  </div>
                  <div className="col-6">
                    <div className="mb-3">
                      <label htmlFor="useremail" className="form-label">
                        Work Item Category
                      </label>
                      <select
                        className="form-select"
                        aria-label="Select example"
                        value={category}
                        onChange={handleChange}
                      >
                        <option value="">Select Work Item Category</option>
                        <option value="Dashboard">Dashboard</option>
                        <option value="Data Quality">Data Quality</option>
                        <option value="Access Management">Access Management</option>
                        <option value="Alerting">Alerting</option>
                        <option value="Workflow">Workflow</option>
                        <option value="Monitoring">Monitoring</option>
                        <option value="Reporting">Reporting</option>
                      </select>
                    </div>
                  </div>
                </div>
                <div className="row">
                  <div className="col-6">
                    <div className="mb-3">
                      <label htmlFor="username" className="form-label">
                        Operational Module
                      </label>
                      <select
                        className="form-select"
                        aria-label="Select example"
                        value={module}
                        onChange={(e) => setModule(e.target.value)}
                      >
                        <option value="">Select module</option>
                        <option value="Dashboard">Dashboard</option>
                        <option value="Data Quality">Data Quality</option>
                        <option value="Workflow">Workflow</option>
                        <option value="Access Management">Access Management</option>
                        <option value="Alerting">Alerting</option>
                      </select>
                    </div>
                  </div>
                  <div className="col-6">
                    <div className="mb-3">
                      <label htmlFor="username" className="form-label">
                        Severity
                      </label>
                      <select
                        className="form-select"
                        aria-label="Select example"
                        value={priority}
                        onChange={(e) => setPriority(e.target.value)}
                      >
                        <option value="">Select severity</option>
                        <option value="High">High</option>
                        <option value="Low">Low</option>
                        <option value="Medium">Medium</option>
                        <option value="Critical">Critical</option>
                      </select>
                    </div>
                  </div>
                </div>

                <div className="row mb-3">
                  <div className="col-12">
                    <label htmlFor="projectdesc-input" className="form-label">
                      Work Item Description / Operational Detail
                    </label>
                    <textarea
                      className="form-control"
                      rows="4"
                      placeholder="Describe the operational issue or request..."
                      value={description}
                      onChange={(e) => setDescription(e.target.value)}
                    />
                  </div>
                </div>
                <div className="row">
                  <div class="col-12">
                    <div class="mb-3">
                      <label for="formFile" class="form-label">Upload Evidence</label>
                      <input class="form-control"
                        type="file"
                        onChange={handleFileChange}
                      />
                    </div>
                  </div>
                </div>
              </div>
              <div className="text-center">
                <button
                  className="btn btn-primary waves-effect waves-light"
                  type="submit"
                  onClick={handleSubmit}
                >
                  {loading ? (
                    <Spinner
                      animation="border"
                      variant="light"
                      role="status"
                      as="span"
                    >
                      <span className="visually-hidden">Loading...</span>
                    </Spinner>
                  ) : (
                    "Add Issue Ticket"
                  )}
                </button>
              </div>
            </div>
          </div>
        </div>
      </div>
    </Fragment>
  );
};

export default AddIssue;
