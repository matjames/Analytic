import React, { Fragment, useState } from 'react'
import { DataGrid, GridRowsProp, GridColDef, GridToolbar } from '@mui/x-data-grid';
import './styles.css'
import AddUser from './AddUser';
import ToolBar from '../../components/ToolBar';
import Card from '../../components/Card';
import Table from '../../components/Table';

const rows: GridRowsProp = [
    { id: 1, col1: 'Hello', col2: 'World', col3: 'Pending' },
    { id: 2, col1: 'DataGridPro', col2: 'is Awesome', col3: 'Pending' },
    { id: 3, col1: 'MUI', col2: 'is Amazing', col3: 'Activated' },
];

const columns: GridColDef[] = [
    { field: 'col1', headerName: 'First Name', width: 150, editable: true },
    { field: 'col2', headerName: 'Last Name', width: 150, editable: true },
    { field: 'col3', headerName: 'Status', width: 150, editable: true },
    {
        field: 'col4', headerName: 'Actions', width: 150, renderCell: (params) => {
            return <div className="actions">
                <div className="edit"><i class='bx bx-edit bxs'></i></div>
                <div className="delete"><i class='bx bx-trash bxs'></i></div>
            </div>
        }
    },
];


const Users = () => {
    const [open, setOpen] = useState(false)
    return (
        <Fragment>
            <ToolBar />
            <div class="content d-flex flex-column flex-column-fluid">
                <div id="kt_content_container" className="container-xxl">
                    <div class="row">
                        <Card />
                        <Card />
                        <Card />
                    </div>
                    <div className="row">
                        <Table />
                    </div>
                </div>
            </div>
        </Fragment>
    )
}

export default Users