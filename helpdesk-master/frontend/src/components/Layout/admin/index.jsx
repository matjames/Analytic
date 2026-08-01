import React from 'react'
import Header from './Header'
import Footer from './Footer'
import TopNavBar from './TopNavBar'

const Layout = ({ children }) => {
    return (
        <div id="layout-wrapper">
            <Header />
            <TopNavBar />
            <div class="main-content">
                <div class="page-content">
                    <div class="container-fluid">
                        {children}
                    </div>
                </div>
                <Footer />
            </div>
        </div>
    )
}

export default Layout