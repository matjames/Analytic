import { saveAs } from "file-saver";
import * as XLSX from "xlsx";

const DownloadExcel = (data, name) => {
  const workbook = XLSX.utils.book_new();

  // Create a new worksheet
  //@ts-ignore
  const worksheet = XLSX.utils.json_to_sheet(data);

  // Add the worksheet to the workbook
  XLSX.utils.book_append_sheet(workbook, worksheet, "Sheet1");

  // Convert the workbook to an Excel file
  const excelFile = XLSX.write(workbook, { bookType: "xlsx", type: "array" });

  // Convert the Excel file to a Blob
  const blob = new Blob([excelFile], {
    type: "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet",
  });

  // Save the Blob as a file
  saveAs(blob, `${name}.xlsx`);
};

export default DownloadExcel;