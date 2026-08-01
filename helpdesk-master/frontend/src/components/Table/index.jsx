import React from 'react'
import DataTable from 'react-data-table-component';

const Table = ({ columns, data, actions, title }) => {
  return (
    <div class="card">
      <div class="card-body">
        <div class="row">
          <div class="col-md-12">
            <div class="card">
              <div class="card-body">
                <h6 class="card-title">{title}</h6>
                <div class="row">
                  <div class="col-sm-12">
                    <DataTable
                      columns={columns}
                      data={data}
                      actions={actions}
                      pagination
                      highlightOnHover
                      selectableRows
                      responsive
                      striped
                      subHeaderAlign="right"
                      subHeaderWrap
                      defaultSortFieldId={1}
                      fixedHeader
                      fixedHeaderScrollHeight="350px"
                      customStyles={{
                        rows: {
                          style: {
                            // minHeight: '56px',
                            fontFamily: 'Roboto',
                            fontSize: '16px',
                          },
                        },
                        headCells: {
                          style: {
                            paddingLeft: '8px',
                            paddingRight: '8px',
                            fontSize: '16px',
                            // fontWeight: 'bold', 
                          },
                        },
                      }}
                    />
                  </div>
                </div>
              </div>
            </div>
          </div>
        </div>
      </div>
    </div>
  )
}

export default Table