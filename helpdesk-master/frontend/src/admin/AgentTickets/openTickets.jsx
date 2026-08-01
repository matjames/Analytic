import React, { useState, useEffect, Fragment } from "react";
import { Link, useHistory } from 'react-router-dom';
import moment from "moment";
import * as XLSX from 'xlsx';
import { saveAs } from 'file-saver';
import API from "../../helpers/api";
import FNModal from "../../components/FNModal";
import FNSpinner from "../../components/FNSpinner";
import FTable from "../SupportTickets/FTable";
import EditIssue from "../SupportTickets/EditIssue";
import Filters from "../../public/Tickets/Filters";

const OpenTickets = () => {
    const [filteredData, setFilteredData] = useState([]);
    const [loading, setLoading] = useState(false);
    const [file, setFile] = useState(null);
    const [id, setId] = useState("");
    const [showEdit, setShowEdit] = useState(false);

    const [currentPage, setCurrentPage] = useState(1);
    const [totalPages, setTotalPages] = useState(1);
    const [totalRecords, setTotalRecords] = useState(0);
    const limit = 10;

    const history = useHistory();
    const closeEdit = () => setShowEdit(false);

    const handleEdit = (id) => {
        setId(id);
        setShowEdit(true);
    };

    const handleFileChange = (e) => {
        setFile(e.target.files[0]);
    };

    const token = localStorage.getItem("token");

    const loadTickets = async (page) => {
        setLoading(true);
        try {

            if (!token) {
                throw new Error("Authorization token not found");
            }

            const res = await API.get(`/t/agents?page=${page}&limit=${limit}`, {
                headers: {
                    Authorization: `Bearer ${token}`
                },
            });

            console.log("Agent ===>", res)
            const formattedTickets = res.data.tickets.map((ticket) => ({
                ...ticket,
                reportedDate: moment(ticket.reportedDate).format("YYYY-MM-DD"),
                status: getStatusBadge(ticket.status),
            }));

            setTotalPages(res?.data.totalPages);
            setTotalRecords(res?.data.totalRecords);
            setFilteredData(formattedTickets);
            setLoading(false);
        } catch (error) {
            console.log("error", error);
            setLoading(false);
        }
    };

    const handlePrevious = () => {
        if (currentPage > 1) {
            setCurrentPage(currentPage - 1);
        }
    };

    const handleNext = () => {
        if (currentPage < totalPages) {
            setCurrentPage(currentPage + 1);
        }
    };

    const getStatusBadge = (status) => {
        switch (status) {
            case "open":
                return <span className="badge bg-danger">{status}</span>;
            case "overdue":
                return <span className="badge bg-warning">{status}</span>;
            case "closed":
                return <span className="badge bg-success">{status}</span>;
            case "inprogress":
                return <span className="badge bg-primary">{status}</span>;
            default:
                return <span className="badge bg-secondary">{status}</span>;
        }
    };

    const handleView = (id) => {
        history.push(`/admin/ticket/${id}`);
    };

    useEffect(() => {
        loadTickets(currentPage);
    }, [currentPage]);

    const downloadExcel = () => {
        const worksheet = XLSX.utils.json_to_sheet(filteredData);
        const workbook = XLSX.utils.book_new();
        XLSX.utils.book_append_sheet(workbook, worksheet, "Tickets");

        const excelBuffer = XLSX.write(workbook, { bookType: 'xlsx', type: 'array' });
        const blob = new Blob([excelBuffer], { type: 'application/octet-stream' });
        saveAs(blob, 'tickets.xlsx');
    };

    return (
        <Fragment>
            <FNModal
                showModal={showEdit}
                handleClose={closeEdit}
                lg="lg"
                title={`Edit Work Item Details - Work Item Id: ${id}`}
            >
                <EditIssue
                    close={closeEdit}
                    refresh={loadTickets}
                    id={id}
                />
            </FNModal>
            <div class="row">
                <div class="col-12">
                    <div class="page-title-b d-sm-flex align-items-center justify-content-between">
                        <h4 class="mb-sm-0 font-size-18">My Open Tickets</h4>
                        <div class="page-title-right">
                            <ol class="breadcrumb m-0">
                                <li class="breadcrumb-item"><Link to="/ict/tickets">Issues</Link></li>
                                <li class="breadcrumb-item active">Reported Issues</li>
                            </ol>
                        </div>
                    </div>
                </div>
            </div>
            <Filters setFilteredData={setFilteredData} fetchData={loadTickets} downloadExcel={downloadExcel} />
            <div class="card">
                <div class="card-body">
                    <div class="row mb-2">
                        <div class="col-sm-4">
                            <div class="search-box me-2 mb-2 d-inline-block">
                                <div class="position-relative">
                                    <input type="text" class="form-control" autocomplete="off" placeholder="Search..." />
                                    <i class="bx bx-search-alt search-icon"></i>
                                </div>
                            </div>
                        </div>
                        <div class="col-sm-8">
                            <div class="text-sm-end">
                                <input id="fileInput" type="file" onChange={handleFileChange} style={{ display: 'none' }} />
                                <label
                                    htmlFor="fileInput"
                                    className="btn btn-primary waves-effect waves-light btn-sm"
                                >
                                    Excel Tickets Bulk Upload <i className="mdi mdi-arrow-right ms-1"></i>
                                </label>
                                {/* <button className="btn btn-primary waves-effect waves-light btn-sm" onClick={handleSubmit}>
                                    Upload Tickets <i class="mdi mdi-arrow-right ms-1"></i>
                                </button> */}
                            </div>
                        </div>
                    </div>
                    <div class="row">
                        <div class="col-12">
                            {loading ? (
                                <FNSpinner />
                            ) : (
                                <FTable data={filteredData} handleView={handleView} handleEdit={handleEdit} />
                            )}
                            <div className="row">
                                <div className="col-sm-12 col-md-5">
                                    <div className="dataTables_info" role="status" aria-live="polite">
                                        Showing {(currentPage - 1) * limit + 1} to {Math.min(currentPage * limit, totalRecords)} of {totalRecords} Records
                                    </div>
                                </div>
                                <div className="col-sm-12 col-md-7">
                                    <div className="dataTables_paginate paging_simple_numbers">
                                        <ul className="pagination">
                                            <li className={`paginate_button page-item previous ${currentPage === 1 ? 'disabled' : ''}`}>
                                                <a onClick={handlePrevious} className="page-link">Previous</a>
                                            </li>
                                            {[...Array(totalPages)].map((_, index) => (
                                                <li key={index} className={`paginate_button page-item ${currentPage === index + 1 ? 'active' : ''}`}>
                                                    <a onClick={() => setCurrentPage(index + 1)} className="page-link">{index + 1}</a>
                                                </li>
                                            ))}
                                            <li className={`paginate_button page-item next ${currentPage === totalPages ? 'disabled' : ''}`}>
                                                <a onClick={handleNext} className="page-link">Next</a>
                                            </li>
                                        </ul>
                                    </div>
                                </div>
                            </div>
                        </div>
                    </div>
                </div>
            </div>
        </Fragment>
    );
};

export default OpenTickets;
