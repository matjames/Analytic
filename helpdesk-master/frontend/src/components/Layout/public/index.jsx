import React, { Fragment } from 'react'
import SideBar from './SideBar'
import Header from './Header'

const Layout = ({ children }) => {
    return (
        <Fragment>
            <SideBar />
                <div class="main-container">
                    <Header />
                    {children}
                </div>
        </Fragment>
    )
}

export default Layout