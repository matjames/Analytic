import React from 'react'
import { Spinner } from "react-bootstrap";

const FNSpinner = () => {
  return (
    <div class="row justify-content-center">
      <Spinner animation="border" variant="primary" role="status" as="span">
        <span className="visually-hidden">Loading...</span>
      </Spinner>
    </div>
  )
}

export default FNSpinner