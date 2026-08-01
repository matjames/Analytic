import React, {Fragment} from 'react';
import {Route, Switch} from 'react-router-dom';
import AdminLayout from '../components/Layout/admin';

import Agents from '../admin/Agents';
import Dashboard from '../admin/Dashboard';
import SupportTickets from '../admin/SupportTickets';
import AddTicket from '../admin/SupportTickets/AddIssue';
import ViewDetails from '../admin/SupportTickets/ViewDetails';
import OpenTickets from '../admin/AgentTickets/openTickets';
import PendingTickets from '../admin/AgentTickets/pendingTickets';
import InProgress from '../admin/AgentTickets/inProgress';
import ClosedTickets from '../admin/AgentTickets/closedTickets';
import OverDue from '../admin/AgentTickets/overDue';
import VideoUpload from '../admin/Videos/VideoUpload';

import KnowledgeBase from "../public/KnowledgeBase";

const AdminRoutes = () => {
    return (
        <Fragment>
            <AdminLayout>
                <Switch>
                    <Route exact path="/admin/agents" component={Agents} />
                    <Route exact path="/admin/dashboard" component={Dashboard} />
                    <Route exact path="/admin/tickets" component={SupportTickets} />
                    <Route exact path="/admin/add/ticket" component={AddTicket} />
                    <Route exact path="/admin/ticket/:id" component={ViewDetails} />
                    <Route exact path="/admin/open/tickets" component={OpenTickets} />
                    <Route exact path="/admin/inprogress/tickets" component={InProgress} />
                    <Route exact path="/admin/closed/tickets" component={ClosedTickets} />
                    <Route exact path="/admin/videos/upload" component={VideoUpload} />
                    <Route exact path="/admin/overdue/tickets" component={OverDue} />
                    <Route exact path="/admin/pending/tickets" component={PendingTickets} />
                    <Route exact path="/admin/knowledge-base" component={KnowledgeBase} />
                </Switch>
            </AdminLayout>
        </Fragment>

    );
};

export default AdminRoutes;