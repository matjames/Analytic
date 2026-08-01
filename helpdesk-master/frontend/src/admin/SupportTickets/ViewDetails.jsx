import React, { useState, useEffect, Fragment } from "react";
import { Link } from 'react-router-dom'
import moment from 'moment';
import API from "../../helpers/api";
import Comments from "../../public/Tickets/Comments";

const ViewDetails = ({ match }) => {

    const [loading, setLoading] = useState(false);
    const [ticket, setTicket] = useState({});

    const { id } = match.params;

    const loadTicket = async () => {
        setLoading(true);
        try {
            const res = await API.get(`/t/tickets/${id}`);
            setTicket(res?.data.ticket);
            setLoading(false);
        } catch (error) {
            console.log("error", error);
            setLoading(false);
        }
    };

    useEffect(() => {
        loadTicket();
    }, []);

    return (
        <Fragment>
            <div class="row">
                <div class="col-12">
                    <div class="pag-title-box d-sm-flex align-items-center justify-content-between mb-3">
                        <h4 class="mb-sm-0 font-size-18">Work Item Number : {ticket && ticket.id}</h4>

                        <div class="page-title-right">
                            <ol class="breadcrumb m-0">
                                <li class="breadcrumb-item"><Link to="/admin/tickets">Back To All Work Items</Link></li>
                                {/* <li class="breadcrumb-item active">Profile</li> */}
                            </ol>
                        </div>

                    </div>
                </div>
            </div>
            <div className="row">
                <div class="col-12">
                    <div class="card">
                        <div class="card-body">
                            <div class="card">
                                <div class="card-body">
                                    <h4 class="card-title mb-4">Work Item Summary</h4>
                                    <p class="text-muted mb-4">{ticket.description}</p>
                                    <h4 class="card-title mb-4">Latest Update</h4>
                                    <p class="text-muted mb-4">{ticket.update ? ticket.update : "No update recorded for this work item yet" }</p>
                                    <div class="table-responsive">
                                        <table class="table table-nowrap mb-0">
                                            <tbody>
                                                <tr>
                                                    <th scope="row">Reported By :</th>
                                                    <td style={{ backgroundColor: '#f2f2f2' }}>{ticket ? ticket.reportedby : ""}</td>
                                                    <th scope="row" >Workstream :</th>
                                                    <td style={{ backgroundColor: '#f2f2f2' }}>{ticket.system || 'Analytics'}</td>
                                                    <th scope="row">Operational Module :</th>
                                                    <td style={{ backgroundColor: '#f2f2f2' }}>{ticket.module || ticket.dhis2module || ticket.emrtype || '-'}</td>
                                                </tr>
                                                <tr>
                                                    <th scope="row">Reported On :</th>
                                                    <td style={{ backgroundColor: '#f2f2f2' }}>{moment(ticket.createdAt).format('YYYY-MM-DD')}</td>
                                                    <th scope="row">Service Area :</th>
                                                    <td style={{ backgroundColor: '#f2f2f2' }}>{ticket.dhis2instance || ticket.emrtype || ticket.facility || '-'}</td>
                                                    <th scope="row">Priority :</th>
                                                    <td style={{ backgroundColor: '#f2f2f2' }}>{ticket.priority}</td>
                                                </tr>
                                                <tr>
                                                    <th scope="row">Phone No :</th>
                                                    <td style={{ backgroundColor: '#f2f2f2' }}>{ticket && ticket.phoneno}</td>
                                                    <th scope="row">Work Item Category :</th>
                                                    <td style={{ backgroundColor: '#f2f2f2' }}>{ticket.category || 'Operational'}</td>
                                                    <th scope="row" >Assigned Operations Lead :</th>
                                                    <td style={{ backgroundColor: '#f2f2f2' }}>{ticket.user ? `${ticket.user.firstname} ${ticket.user.lastname}` : ""}</td>
                                                </tr>
                                                <tr>
                                                    <th scope="row">Facility :</th>
                                                    <td style={{ backgroundColor: '#f2f2f2' }}>{ticket && ticket.facility}</td>
                                                    <th scope="row">Status :</th>
                                                    <td style={{ backgroundColor: '#f2f2f2' }}>{ticket.status}</td>
                                                    <th scope="row">Due Date :</th>
                                                    <td style={{ backgroundColor: '#f2f2f2' }}>{ticket.dueDate}</td>
                                                </tr>
                                            </tbody>
                                        </table>
                                    </div>
                                </div>
                                <Comments id={id} />
                            </div>
                        </div>
                    </div>
                </div>
            </div>

            
        </Fragment>
    )
}

export default ViewDetails