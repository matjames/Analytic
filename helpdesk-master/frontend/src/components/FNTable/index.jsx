import React, { useState, useEffect } from 'react';
import Select from 'react-select';

const FNTable = ({ columns, data, onViewDetails }) => {
    const [filters, setFilters] = useState({});

    useEffect(() => {
        const initialFilters = columns.reduce((acc, column) => {
            acc[column.key] = null;
            return acc;
        }, {});
        setFilters(initialFilters);
    }, [columns]);

    const handleFilterChange = (key, selectedOption) => {
        setFilters(prevFilters => ({
            ...prevFilters,
            [key]: selectedOption
        }));
    };

    const filteredData = data.filter(row => {
        return columns.every(column => {
            if (!filters[column.key]) return true;
            return row[column.key] === (filters[column.key] ? filters[column.key].value : null);
        });
    });
useEffect(()=>{console.log("filteredData", filteredData)},[])
    return (
        <div className="table-responsive ">
            <table className="table align-middle table-nowrap dt-responsive nowrap w-100 vh-100" id="customerList-table">
                <thead className="table-light">
                    <tr>
                        <th style={{ width: '10px' }}>
                            <div className="form-check font-size-16 align-middle">
                                <input className="form-check-input" type="checkbox" id="transactionCheck01" />
                                <label className="form-check-label" htmlFor="transactionCheck01"></label>
                            </div>
                        </th>
                        {columns.map((column, index) => (
                            <th key={index} className="align-middle">
                                <div style={{ display: 'flex', alignItems: 'center' }}>
                                    {column.label}
                                    {column.filterOptions && (
                                        <Select
                                            options={column.filterOptions}
                                            value={filters[column.key]}
                                            onChange={(selectedOption) => handleFilterChange(column.key, selectedOption)}
                                            placeholder=""
                                            isClearable
                                            className="ms-2"
                                            styles={{
                                                control: (base) => ({
                                                    ...base,
                                                    minHeight: '1.5rem',
                                                    height: '1.5rem',
                                                    width: '150px',
                                                }),
                                                dropdownIndicator: (base) => ({
                                                    ...base,
                                                    padding: '0px 8px',
                                                }),
                                                clearIndicator: (base) => ({
                                                    ...base,
                                                    padding: '0px 8px',
                                                }),
                                                valueContainer: (base) => ({
                                                    ...base,
                                                    padding: '0px 6px',
                                                }),
                                                input: (base) => ({
                                                    ...base,
                                                    margin: '0px',
                                                }),
                                                indicatorsContainer: (base) => ({
                                                    ...base,
                                                    height: '1.2rem',
                                                }),
                                            }}
                                        />
                                    )}
                                </div>
                            </th>
                        ))}
                        <th className="align-middle">View Details</th>
                    </tr>
                </thead>
                <tbody className="">
                    {filteredData.map((row, rowIndex) => (
                        <tr key={rowIndex}>
                            <td>
                                <div className="form-check font-size-16">
                                    <input className="form-check-input" type="checkbox" id={`transactionCheck${rowIndex + 2}`} />
                                    <label className="form-check-label" htmlFor={`transactionCheck${rowIndex + 2}`}></label>
                                </div>
                            </td>
                            {columns.map((column, colIndex) => (
                                <td key={colIndex}>{row[column.key]}</td>
                            ))}
                            <td>
                                <button
                                    type="button"
                                    className="btn btn-primary btn-sm btn-rounded waves-effect waves-light"
                                    onClick={() => onViewDetails(row.id)}
                                >
                                    View Details
                                </button>
                            </td>
                        </tr>
                    ))}
                </tbody>
            </table>
        </div>
    );
};

export default FNTable;
