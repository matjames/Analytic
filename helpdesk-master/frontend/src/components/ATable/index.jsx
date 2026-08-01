/* eslint-disable jsx-a11y/anchor-is-valid */
import React from "react";

const ATable = ({
  columns,
  data,
  onViewDetails,
  text,
  handleEdit,
  handleDelete,
}) => {
  const getValue = (obj, path) => {
    const keys = path.split(".");
    return keys.reduce((acc, key) => acc && acc[key], obj);
  };

  return (
    <div class="table-responsive">
      <table
        class="table align-middle table-nowrap dt-responsive nowrap w-100"
        id="customerList-table"
      >
        <thead class="table-light">
          <tr>
            <th style={{ width: "20px" }}>
              <div className="form-check font-size-16 align-middle">
                <input
                  className="form-check-input"
                  type="checkbox"
                  id="transactionCheck01"
                />
                <label
                  className="form-check-label"
                  htmlFor="transactionCheck01"
                ></label>
              </div>
            </th>
            {columns.map((column, index) => (
              <th key={index} className="align-middle">
                {column.label}
              </th>
            ))}
            <th className="align-middle">Actions</th>
          </tr>
        </thead>
        <tbody>
          {data.map((row, rowIndex) => (
            <tr key={rowIndex}>
              <td>
                <div className="form-check font-size-16">
                  <input
                    className="form-check-input"
                    type="checkbox"
                    id={`transactionCheck${rowIndex + 2}`}
                  />
                  <label
                    className="form-check-label"
                    htmlFor={`transactionCheck${rowIndex + 2}`}
                  ></label>
                </div>
              </td>
              {/* {columns.map((column, colIndex) => (
                                <td key={colIndex}>{row[column.key]}</td>
                            ))} */}
              {columns.map((column, colIndex) => (
                <td key={colIndex}>{getValue(row, column.key)}</td> // Use getValue function to handle nested properties
              ))}
              <td>
                <div
                  class="d-inline-block me-2"
                  data-bs-toggle="tooltip"
                  data-bs-placement="top"
                >
                  <a
                    onClick={() => onViewDetails(row.id)}
                    class="action-icon text-primary"
                    style={{ cursor: "pointer" }}
                  >
                    <i class="mdi mdi-eye font-size-20"></i>
                  </a>
                </div>
                <div
                  class="d-inline-block me-2"
                  data-bs-toggle="tooltip"
                  data-bs-placement="top"
                >
                  <a
                    onClick={() => handleEdit(row)}
                    class="action-icon text-warning"
                    style={{ cursor: "pointer" }}
                  >
                    <i class="mdi mdi-comment-edit-outline font-size-20"></i>
                  </a>
                </div>
                <div
                  class="d-inline-block me-2"
                  data-bs-toggle="tooltip"
                  data-bs-placement="top"
                >
                  <a
                    onClick={() => handleDelete(row.id)}
                    class="action-icon text-danger"
                    style={{ cursor: "pointer" }}
                  >
                    <i class="mdi mdi-trash-can font-size-20"></i>
                  </a>
                </div>
              </td>
            </tr>
          ))}
        </tbody>
      </table>
    </div>
  );
};

export default ATable;
