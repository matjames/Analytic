import React, { useState, useEffect, Fragment } from 'react'
import { toast } from "react-toastify";
import Select from 'react-select';
import API from "../../helpers/api";
import { Spinner } from "react-bootstrap";

const EditIssue = ({ id, refresh, close }) => {

  const [ticket, setTicket] = useState("");
  const [agentId, setAgentId] = useState("");
  const [update, setUpdate] = useState("");
  const [status, setStatus] = useState("");
  const [agents, setAgents] = useState([]);
  const [loading, setLoading] = useState(false);

  const loadIssue = async () => {
    setLoading(true);
    try {
      const res = await API.get(`/t/tickets/${id}`);
      console.log("response ===>", res)
      setTicket(res.data.ticket);
      setStatus(res.data.ticket.status);
      setAgentId(res.data.ticket.agentId);
      setLoading(false);
    } catch (error) {
      setLoading(false);
    }
  };

  const loadAgents = async () => {
    setLoading(true);
    try {
      const res = await API.get(`/users`);
      setAgents(res.data.users)
      setLoading(false);
    } catch (error) {
      setLoading(false);
    }
  };

  const handleChange = (selectedOption) => {
    setAgentId(selectedOption.value);
  };

  const handleSubmit = async (e) => {
    e.preventDefault();
    setLoading(true);

    const data = {
      status,
      agentId,
      update
    };

    try {
      const response = await API.patch(`/t/tickets/${id}`, data,);
      setLoading(false);
      refresh();
      close();
      toast.success("Work Item Details Updated Successfully !!");
    } catch {
      setLoading(false);
      toast.error("Error while Updating Work Item Details");
    }
  };

  useEffect(() => {
    loadIssue();
    loadAgents();
  }, []);

  return (
    <Fragment>
      <div class="row">
        <div class="card overflow-hidden">
          <div class="card-body pt-0">
            <div class="p-4">
              <div className="row">
                <div className="col-6">
                  <div class="mb-3">
                    <label for="useremail" class="form-label">Update Status</label>
                    <select class="form-select" aria-label="Select example"
                      value={status}
                      onChange={(e) => setStatus(e.target.value)}
                    >
                      <option value="">Select Ticket Status</option>
                      <option value="open">Open</option>
                      <option value="inprogress">In Progress</option>
                      <option value="closed">Closed</option>
                      <option value="overdue">Over Due</option>
                    </select>
                  </div>
                </div>
                <div className="col-6">
                  <div class="mb-3">
                    <label for="username" class="form-label">Assign Agent</label>
                    <Select
                      defaultValue={agentId}
                      onChange={handleChange}
                      options={agents.map(agent => ({ value: agent.id, label: agent.username }))}
                      placeholder="Assign Agent"
                    />
                  </div>
                </div>
              </div>
              <div className="row mb-2">
                <div className="col-12">
                  <label for="projectdesc-input" class="form-label">Work Item Description</label>
                  <textarea class="form-control" rows="2" placeholder="Describe the work item..."
                    value={ticket && ticket.description}
                    disabled
                  />
                </div>
              </div>
              <div className="row mb-2">
                <div className="col-12">
                  <label for="projectdesc-input" class="form-label">Latest Update</label>
                  <textarea class="form-control" rows="2" placeholder="Enter the latest update..."
                    value={update}
                    onChange={(e) => setUpdate(e.target.value)}
                  />
                </div>
              </div>
            </div>
            <div class="text-center">
              <button class="btn btn-primary waves-effect waves-light" type="submit" onClick={handleSubmit}>
                {loading ? <Spinner animation="border" variant="light" role="status" as="span">
                  <span className="visually-hidden">Loading...</span>
                </Spinner> : "Update Work Item Details"}
              </button>
            </div>
          </div>
        </div>
      </div>
    </Fragment>

  )
}

export default EditIssue