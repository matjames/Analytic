import React, { useState } from 'react';
import Select from "react-select";
import { Spinner } from "react-bootstrap";
import API from "../../helpers/api";
import healthFacilities from "../../data/facilities";

const Filters = ({ setFilteredData, fetchData, downloadExcel }) => {

    const [level, setLevel] = useState("");
    const [facility, setFacility] = useState("");
    const [status, setStatus] = useState("");
    const [system, setSystem] = useState("");
    const [category, setCategory] = useState("");
    const [loading, setLoading] = useState(false);

    const filteredHealthFacilities = healthFacilities.filter(
        (facility) => facility.level === level
    );

    const handleSelectChange = (selectedOption) => setFacility(selectedOption);

    const handleFilterChange = async () => {
        setLoading(true);
        try {
            const response = await API.get('/t/tickets', {
                params: {
                    category,
                    status,
                    system,
                    level,
                    facility: facility ? facility.value : '',
                },
            });
            setFilteredData(response.data.tickets);
        } catch (error) {
            console.error(error);
        } finally {
            setLoading(false);
        }
    };

    return (
        <div class="row">
            <div class="col-lg-12">
                <div class="card job-filter">
                    <div class="card-body p-3">
                        <div class="row">
                            <div class="col-2">
                                <div class="position-relative">
                                    <select class="form-select"
                                        value={category}
                                        onChange={(e) => setCategory(e.target.value)}>
                                        <option value="">Select Category</option>
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
                            <div class="col-2">
                                <div class="position-relative">
                                    <select class="form-select"
                                        value={status}
                                        onChange={(e) => setStatus(e.target.value)}>
                                        <option value="">Status</option>
                                        <option value="Open">Open</option>
                                        <option value="In Progress">In Progress</option>
                                        <option value="Resolved">Resolved</option>
                                        <option value="Closed">Closed</option>
                                    </select>
                                </div>
                            </div>
                            <div class="col-2">
                                <div class="position-relative">
                                    <div id="datepicker1">
                                        <select class="form-select"
                                            value={system}
                                            onChange={(e) => setSystem(e.target.value)}>
                                            <option value="">Select Workstream</option>
                                            <option value="Analytics">Analytics</option>
                                            <option value="Data Pipeline">Data Pipeline</option>
                                            <option value="Governance">Governance</option>
                                            <option value="Reporting">Reporting</option>
                                            <option value="Operations">Operations</option>
                                        </select>
                                    </div>
                                </div>
                            </div>
                            <div class="col-2">
                                <div class="position-relative">
                                    <select
                                        className="form-select"
                                        aria-label="Select example"
                                        value={level}
                                        onChange={(e) => setLevel(e.target.value)}
                                    >
                                        <option value="">Select Facility Level</option>
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
                            <div class="col-2">
                                <div class="position-relative">
                                    <Select
                                        options={filteredHealthFacilities}
                                        value={facility}
                                        onChange={handleSelectChange}
                                        placeholder="Select Service Area"
                                        isSearchable
                                    />
                                </div>
                            </div>
                            <div class="col-2">
                                <div class="position-relative h-100 hstack gap-3">
                                    <div class="flex-shrink-0">
                                        <a href="#!" onClick={handleFilterChange} class="btn btn-primary">
                                            <i class="mdi mdi-filter-outline align-middle"></i>Filters</a>
                                        <a href="#!" onClick={fetchData} class="btn btn-light mx-2"><i class="mdi mdi-refresh"></i> Clear</a>
                                        <a href="#!" onClick={downloadExcel} class="btn btn-success"><i class="bx bx-down-arrow-alt"></i> </a>
                                    </div>
                                </div>
                            </div>
                        </div>
                    </div>
                </div>
            </div>
        </div>
    )
}

export default Filters