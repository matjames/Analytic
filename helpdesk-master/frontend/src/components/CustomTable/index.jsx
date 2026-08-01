import React, { useState, useRef } from "react";
import { Input, Button, Space, Table } from "antd";
import { SearchOutlined } from "@ant-design/icons";
import "./table.css";

const CustomTable = ({ columns, data }) => {
  const [searchText, setSearchText] = useState("");
  const [searchedColumn, setSearchedColumn] = useState("");
  const searchInput = useRef(null);
  const [searchAll, setSearchAll] = useState("");

  const handleSearch = (selectedKeys, confirm, dataIndex) => {
    confirm();
    setSearchText(selectedKeys[0]);
    setSearchedColumn(dataIndex);
  };

  const handleReset = (clearFilters) => {
    clearFilters();
    setSearchText("");
  };

  const getColumnSearchProps = (dataIndex) => ({
    filterDropdown: ({
      setSelectedKeys,
      selectedKeys,
      confirm,
      clearFilters,
    }) => (
      <div style={{ padding: 8 }}>
        <Input
          ref={searchInput}
          placeholder={`Search ${dataIndex}`}
          value={selectedKeys[0]}
          onChange={(e) =>
            setSelectedKeys(e.target.value ? [e.target.value] : [])
          }
          onPressEnter={() => handleSearch(selectedKeys, confirm, dataIndex)}
          style={{ width: 188, marginBottom: 8, display: "block" }}
        />
        <Space>
          <Button
            type="primary"
            onClick={() => handleSearch(selectedKeys, confirm, dataIndex)}
            icon={<SearchOutlined />}
            size="small"
            style={{ width: 90 }}
          >
            Search
          </Button>
          <Button
            onClick={() => handleReset(clearFilters)}
            size="small"
            style={{ width: 90 }}
          >
            Reset
          </Button>
        </Space>
      </div>
    ),
    filterIcon: (filtered) => (
      <SearchOutlined style={{ color: filtered ? "#1890ff" : undefined }} />
    ),
    onFilter: (value, record) =>
      record[dataIndex]
        ? record[dataIndex]
            .toString()
            .toLowerCase()
            .includes(value.toLowerCase())
        : "",
    onFilterDropdownVisibleChange: (visible) => {
      if (visible) {
        setTimeout(() => searchInput.current.select(), 100);
      }
    },
    render: (text) =>
      searchedColumn === dataIndex ? (
        <span style={{ fontWeight: "bold" }}>{text}</span>
      ) : (
        text
      ),
  });

  const modifiedColumns = columns.map((column) => {
    if (column.dataIndex) {
      return {
        ...column,
        ...getColumnSearchProps(column.dataIndex),
      };
    }
    return column;
  });

  const tableHeaderStyle = {
    background: "#f0f0f0", // Example background color
    fontWeight: "bold",
  };
  const searchedData = data.filter((item) =>
    searchAll.toLowerCase() === ""
      ? true
      : item.issueCategory.toLowerCase().includes(searchAll.toLowerCase())
  );

  return (
    <div className="row ">
      <div className="col-md-12">
        <div className="row">
          <div class="col-sm-4">
            <div class="search-box me-2 mb-2 d-inline-block">
              <div class="position-relative">
                <input
                  type="text"
                  class="form-control"
                  onChange={(e) => setSearchAll(e.target.value)}
                  placeholder="Search...."
                />
                <i class="bx bx-search-alt search-icon"></i>
              </div>
            </div>
          </div>
        </div>
      </div>
      <div className="col-md-12">
        <div className="card">
          <div className="card-body">
            <div className="row">
              <div className="col-sm-12 shadow-sm">
                <Table
                  columns={modifiedColumns}
                  dataSource={searchedData}
                  className="custom-table"
                  style={{ borderRadius: 8 }} // Example: Add border radius
                  bordered
                  headerClassName="custom-header"
                />
              </div>
            </div>
          </div>
        </div>
      </div>
    </div>
  );
};

export default CustomTable;
