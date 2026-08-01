import React from 'react'
import { Modal } from "react-bootstrap";

const FNModal = ({ children, showModal, handleClose, lg, title }) => {
    return (
        <Modal show={showModal} onHide={handleClose} size={lg} backdrop="static" keyboard={false}>
            <Modal.Header closeButton style={{ backgroundColor: "#556ee6" }}>
                <Modal.Title className='text-white'>{title}</Modal.Title>
            </Modal.Header>
            <Modal.Body>{children}</Modal.Body>
        </Modal>
    )
}

export default FNModal