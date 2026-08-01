import React from 'react';
import { Route, Switch } from 'react-router-dom';
import PublicLayout from '../components/Layout/public';

import FAQ from "../public/FAQ"
import Issues from '../public/Issues';
import Overview from '../public/Overview';
import Tickets from '../public/Tickets';
import TicketDetails from '../public/Tickets/TicketDetails';
import Videos from '../public/Videos';
import Documentation from '../public/Documentation';
import KnowledgeBase from '../public/KnowledgeBase';
import KBCategory from '../public/KnowledgeBase/KBCategory';
import ArticleDetails from '../public/KnowledgeBase/ArticleDetails';
import Kaban from '../public/KabanBoard';

const PublicRoutes = () => {
    return (
        <PublicLayout>
            <Switch>
                <Route exact path="/public" component={Overview} />
                <Route exact path="/public/submit" component={Issues} />
                <Route exact path="/public/videos" component={Videos} />
                <Route exact path="/public/faq" component={FAQ} />
                <Route exact path="/public/tickets" component={Tickets} />
                <Route exact path="/public/kaban" component={Kaban} />
                <Route exact path="/public/ticket/details" component={TicketDetails} />
                <Route exact path="/public/kb/article" component={ArticleDetails} />
                <Route exact path="/public/kb/category" component={KBCategory} />
                <Route exact path="/public/knowledgeBase" component={KnowledgeBase} />
                <Route exact path="/public/documentation" component={Documentation} />
            </Switch>
        </PublicLayout>
    );
};

export default PublicRoutes;