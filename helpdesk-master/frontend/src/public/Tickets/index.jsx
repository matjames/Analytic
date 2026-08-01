import React, { useState, useEffect, Fragment } from "react";
import { useHistory } from 'react-router-dom';
import moment from "moment";
import * as XLSX from 'xlsx';
import { saveAs } from 'file-saver';
import API from "../../helpers/api";
import './styles.css';

const Tickets = () => {
  const [filteredData, setFilteredData] = useState([]);
  const [loading, setLoading] = useState(false);
  const [currentPage, setCurrentPage] = useState(1);
  const [totalPages, setTotalPages] = useState(1);
  const [totalRecords, setTotalRecords] = useState(0);
  const limit = 10;

  const history = useHistory();

  const loadTickets = async (page) => {
    setLoading(true);
    try {
      const res = await API.get(`/t/tickets?page=${page}&limit=${limit}`);
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

  const handleView = () => {
    history.push(`/public/ticket/details/`);
  };

  useEffect(() => {
    loadTickets(currentPage);
  }, [currentPage]);

  const downloadExcel = () => {
    const worksheet = XLSX.utils.json_to_sheet(filteredData);
    const workbook = XLSX.utils.book_new();
    XLSX.utils.book_append_sheet(workbook, worksheet, "Operations Work Items");

    const excelBuffer = XLSX.write(workbook, { bookType: 'xlsx', type: 'array' });
    const blob = new Blob([excelBuffer], { type: 'application/octet-stream' });
    saveAs(blob, 'operations-work-items.xlsx');
  };

  return (
    <Fragment>
      <div class="content-area">
        <div class="tickets-header">
          <div class="tickets-title">
            <h1>
              <i class="fas fa-star" style={{ color: '#f39c12' }}></i>
              Operations Work Items
              <span class="ticket-count">04</span>
            </h1>
            <div class="tickets-controls">
              <button class="control-btn">
                <i class="fas fa-filter"></i>
              </button>
              <button class="control-btn">
                <i class="fas fa-sync-alt"></i>
              </button>
            </div>
          </div>
          <div class="tickets-actions">
            <a href="#" class="total-count">Workspace View</a>
            <div class="view-selector">
              <i class="fas fa-list"></i>
              <span>Operations View</span>
            </div>
            <button class="control-btn" onClick={downloadExcel}>
              <i class="fas fa-download"></i>
            </button>
          </div>
        </div>

        <div class="tickets-list">
          <div class="ticket-item" style={{ cursor: 'pointer' }} onClick={() => handleView()}>
            <div class="ticket-icon">
              <i class="fas fa-exclamation-triangle"></i>
            </div>
            <div class="ticket-content">
              <a href="#" class="ticket-title">Dashboard refresh failed for district reports</a>
              <div class="ticket-meta">
                <span>#101</span>
                <span>Grace A.</span>
                <span>Analytics</span>
                <span><i class="far fa-clock"></i> 02:20 PM</span>
                <span><i class="far fa-calendar"></i> Today</span>
              </div>
            </div>
            <div class="ticket-actions">
              <div class="ticket-status">Open</div>
              <div class="ticket-assignee">GA</div>
            </div>
          </div>

          <div class="ticket-item" style={{ cursor: 'pointer' }} onClick={() => handleView()}>
            <div class="ticket-icon">
              <i class="fas fa-envelope"></i>
            </div>
            <div class="ticket-content">
              <a href="#" class="ticket-title">Weekly data refresh delayed by 45 minutes</a>
              <div class="ticket-meta">
                <span>#100</span>
                <span>John K.</span>
                <span>Data Pipeline</span>
                <span><i class="far fa-clock"></i> 02:12 PM</span>
                <span><i class="far fa-calendar"></i> Today</span>
              </div>
            </div>
            <div class="ticket-actions">
              <div class="ticket-status">In Progress</div>
              <div class="ticket-assignee">JK</div>
            </div>
          </div>

          <div class="ticket-item" style={{ cursor: 'pointer' }} onClick={() => handleView()}>
            <div class="ticket-icon">
              <i class="fas fa-exclamation-triangle"></i>
            </div>
            <div class="ticket-content">
              <a href="#" class="ticket-title">Access request pending for a new analyst</a>
              <div class="ticket-meta">
                <span>#99</span>
                <span>Sarah N.</span>
                <span>Access Management</span>
                <span><i class="far fa-clock"></i> 11:40 AM</span>
                <span><i class="far fa-calendar"></i> Today</span>
              </div>
            </div>
            <div class="ticket-actions">
              <div class="ticket-status">Pending</div>
              <div class="ticket-assignee">SN</div>
            </div>
          </div>

          <div class="ticket-item" style={{ cursor: 'pointer' }} onClick={() => handleView()}>
            <div class="ticket-icon">
              <i class="fas fa-envelope"></i>
            </div>
            <div class="ticket-content">
              <a href="#" class="ticket-title">Alert threshold update required for weekly monitoring</a>
              <div class="ticket-meta">
                <span>#98</span>
                <span>Grace A.</span>
                <span>Alerting</span>
                <span><i class="far fa-clock"></i> 09:10 AM</span>
                <span><i class="far fa-calendar"></i> Yesterday</span>
              </div>
            </div>
            <div class="ticket-actions">
              <div class="ticket-status">Resolved</div>
              <div class="ticket-assignee">GA</div>
            </div>
          </div>
        </div>
      </div>
    </Fragment>
  );
};

export default Tickets;