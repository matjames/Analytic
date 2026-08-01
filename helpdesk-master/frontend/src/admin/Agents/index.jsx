import React, { useState, useEffect, Fragment } from "react";
import { Link } from "react-router-dom";
import moment from "moment";
import { toast } from "react-toastify";
import FNModal from "../../components/FNModal";
import API from "../../helpers/api";
import FNSpinner from "../../components/FNSpinner";
import ATable from "../../components/ATable";
import AddAgent from "./AddAgent";
import EditAgent from "./EditAgent";

const Agents = () => {
  const [loading, setLoading] = useState(false);
  const [agents, setAgents] = useState([]);
  const [id, setId] = useState("");
  const [showEdit, setShowEdit] = useState(false);
  const [showModal, setShowModal] = useState(false);
  const [showOpen, setShowOpen] = useState(false);
  const [singleAgent, setSingleAgent] = useState({});

  const [currentPage, setCurrentPage] = useState(1);
  const [totalPages, setTotalPages] = useState(1);
  const [totalRecords, setTotalRecords] = useState(0);
  const limit = 10;

  const handleShow = () => setShowModal(true);
  const handleClose = () => setShowModal(false);
  const closeEdit = () => setShowEdit(false);

  const show = () => setShowOpen(true);
  const close = () => setShowOpen(false);

  const handleEdit = (row) => {
    setId(row.id);
    setShowEdit(true);
    setSingleAgent(row);
  };

  const handleView = (id) => {
    handleShow();
    setId(id);
  };

  const handleDelete = async (id) => {
    setId(id);
    setLoading(true);
    try {
      const res = await API.delete(`/user/${id}`);
      loadAgents();
      setLoading(false);
      toast.success(`Agent Has Been Deleted Successfully`);
    } catch (error) {
      console.log("error", error);
      setLoading(false);
      toast.error("Error while Deleting Agent Details");
    }
  };

  const loadAgents = async (page) => {
    setLoading(true);
    try {
      const res = await API.get(`/users/agents?page=${page}&limit=${limit}`);
      const users = res?.data.users.map((spare) => ({
        ...spare,
        createdAt: moment(spare.createdAt).format("YYYY-MM-DD"),
      }));

      setAgents(users);
      setTotalPages(res?.data.totalPages);
      setTotalRecords(res?.data.totalRecords);
      setLoading(false);
    } catch (error) {
      console.log("error", error);
      setLoading(false);
    }
  };

  

  const tableColumns = [
    { key: "username", label: "User Name" },
    { key: "firstname", label: "First Name" },
    { key: "lastname", label: "Last Name" },
    { key: "phoneNo", label: "Phone Number" },
    { key: "email", label: "Email" },
    { key: "system", label: "Operations System" },
    { key: "createdAt", label: "Reported Date" },
  ];

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

  useEffect(() => {
    loadAgents(currentPage);
  }, [currentPage]);

  return (
    <Fragment>
      <FNModal
        showModal={showOpen}
        handleClose={close}
        lg="lg"
        title="Add New Operations Agent"
      >
        <AddAgent close={close} refresh={loadAgents} id={id} />
      </FNModal>
      <FNModal
        showModal={showEdit}
        handleClose={closeEdit}
        lg="lg"
        title="Edit Operations Agent Details"
      >
        <EditAgent
          close={closeEdit}
          refresh={loadAgents}
          id={id}
          singleAgent={singleAgent}
        />
      </FNModal>
      <div class="row">
        <div class="col-12">
          <div class="mb-3 d-sm-flex align-items-center justify-content-between">
            <h4 class="mb-sm-0 font-size-18">Operations Agents</h4>
            <div class="page-title-right">
              <ol class="breadcrumb m-0">
                <li class="breadcrumb-item">
                  <Link to="/ict/assets">Operations Agents</Link>
                </li>
                <li class="breadcrumb-item active">Listing</li>
              </ol>
            </div>
          </div>
        </div>
      </div>
      {loading ? (
        <FNSpinner />
      ) : (
        <>
          <div class="card">
            <div class="card-body">
              <div class="row">
                <div class="col-12">
                  <div class="card">
                    <div class="card-body">
                      <div class="row mb-2">
                        <div class="col-sm-4">
                          <div class="search-box me-2 mb-2 d-inline-block">
                            <div class="position-relative">
                              <input
                                type="text"
                                class="form-control"
                                id="searchTableList"
                                placeholder="Search..."
                              />
                              <i class="bx bx-search-alt search-icon"></i>
                            </div>
                          </div>
                        </div>
                        <div class="col-sm-8">
                          <div class="text-sm-end">
                            <button
                              type="submit"
                              class="btn btn-primary waves-effect waves-light"
                              onClick={show}
                            >
                              Add New Agent
                            </button>
                          </div>
                        </div>
                      </div>
                      <ATable
                        columns={tableColumns}
                        data={agents}
                        handleEdit={handleEdit}
                        onViewDetails={handleView}
                        handleDelete={handleDelete}
                      />
                      <div class="row">
                        <div class="col-12">
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
                </div>
              </div>
            </div>
          </div>
        </>
      )}
    </Fragment>
  );
};

export default Agents;
