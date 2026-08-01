import React, { Fragment } from 'react';
import { Route, Switch, Redirect } from "react-router-dom";
import { ToastContainer } from "react-toastify";
import "react-toastify/dist/ReactToastify.css";

import Auth from '../admin/Auth/Login';
import AdminRoutes from './AdminRoutes';
import PublicRoutes from './PublicRoutes';

const Routes = () => {
    return (
        <Fragment>
            <ToastContainer />
            <Route exact path="/login" component={Auth} />
            <Switch>
                <Route path="/public" component={PublicRoutes} />
                <Route path="/admin" component={AdminRoutes} />
                <Route path="/" exact>
                    <Redirect to="/public" />
                </Route>
            </Switch>
        </Fragment>
    );
};

export default Routes;