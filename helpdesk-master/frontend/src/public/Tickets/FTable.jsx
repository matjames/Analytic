/* eslint-disable jsx-a11y/anchor-is-valid */
/* eslint-disable no-undef */
import React from 'react'

const FTable = ({ data, handleView, handleEdit, handleDelete }) => {

    const truncateText = (text, maxLength) => {
        if (text.length <= maxLength) {
          return text;
        }
        return text.substring(0, maxLength) + '...';
      };

    return (
        <div class="table-responsive">
            <table class="table align-middle table-striped table-sm">
                <thead className="table-dark">
                    <tr>
                        <th>Work Item</th>
                        <th>Category</th>
                        <th>Workstream</th>
                        <th>Module / Context</th>
                        <th>Summary</th>
                        <th>Status</th>
                        <th>Service Area</th>
                        <th>Actions</th>
                    </tr>
                </thead>
                <tbody>
                    {data &&
                        data.map((item) => (
                            <tr key={item.id}>
                                <td>{item.id}</td>
                                <td>{item.category || 'Operational'}</td>
                                <td>{item.system || 'Analytics'}</td>
                                <td>{item.module || item.emrtype || item.dhis2instance || item.dhis2module || '-'}</td>
                                <td>{truncateText(item.description, 70)}</td>
                                <td>{item.status}</td>
                                <td>{item.facility || 'Unassigned'}</td>
                                <td>
                                    <div class="d-inline-block me-2">
                                        <a onClick={() => handleView(item.id)} class="action-icon text-primary" style={{ cursor: 'pointer' }}>
                                            <i class="mdi mdi-eye font-size-20"></i></a>
                                    </div>
                                    {/* <div class="d-inline-block me-2">
                                        <a onClick={() => handleEdit(item.id)} class="action-icon text-warning" style={{ cursor: 'pointer' }}>
                                            <i class="mdi mdi-comment-edit-outline font-size-20"></i></a>
                                    </div> */}
                                    {/* <div class="d-inline-block me-2">
                                        <a onClick={() => handleDelete(item.id)} class="action-icon text-danger" style={{ cursor: 'pointer' }}>
                                            <i class="mdi mdi-trash-can font-size-20"></i></a>
                                    </div> */}
                                </td>
                            </tr>
                        ))}
                </tbody>
            </table>
        </div>
    )
}

export default FTable