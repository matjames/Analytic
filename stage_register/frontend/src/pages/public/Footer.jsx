import React from 'react'

const Footer = () => {
    const currentYear = new Date().getFullYear();
    
    return (
        <footer className="footer">
            <p>&copy; {currentYear} StatGate Analytical Orchestration Engine – Field Operations & Agent Registry. All rights reserved.</p>
        </footer>
    )
}

export default Footer