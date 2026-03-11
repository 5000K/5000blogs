(function() {
  'use strict';

  function initSortableTables() {
    const tables = document.querySelectorAll('table');

    tables.forEach(table => {
      const headerRow = findHeaderRow(table);
      if (!headerRow) return;

      const headers = headerRow.querySelectorAll('th');
      if (headers.length === 0) return;

      const tbody = table.querySelector('tbody') || table;
      const originalOrder = Array.from(tbody.querySelectorAll('tr'))
        .filter(row => row !== headerRow);

      headers.forEach((header, columnIndex) => {
        let sortState = 'none'; 

        header.style.cursor = 'pointer';

        header.addEventListener('click', () => {

          headers.forEach(h => {
            h.classList.remove('sort-tables-plugin-asc', 'sort-tables-plugin-desc');
          });

          if (sortState === 'none') {
            sortState = 'asc';
            header.classList.add('sort-tables-plugin-asc');
            sortTable(tbody, headerRow, columnIndex, 'asc');
          } else if (sortState === 'asc') {
            sortState = 'desc';
            header.classList.add('sort-tables-plugin-desc');
            sortTable(tbody, headerRow, columnIndex, 'desc');
          } else {
            sortState = 'none';
            restoreOriginalOrder(tbody, headerRow, originalOrder);
          }
        });
      });
    });
  }

  function findHeaderRow(table) {
    const thead = table.querySelector('thead');
    if (thead) {
      const theadRow = thead.querySelector('tr');
      if (theadRow && theadRow.querySelectorAll('th').length > 0) {
        return theadRow;
      }
    }

    const firstRow = table.querySelector('tr');
    if (firstRow && firstRow.querySelectorAll('th').length > 0) {
      return firstRow;
    }

    return null;
  }

  function sortTable(tbody, headerRow, columnIndex, direction) {
    const rows = Array.from(tbody.querySelectorAll('tr'))
      .filter(row => row !== headerRow);

    rows.sort((rowA, rowB) => {
      const cellA = rowA.children[columnIndex];
      const cellB = rowB.children[columnIndex];

      if (!cellA || !cellB) return 0;

      const valueA = getCellValue(cellA);
      const valueB = getCellValue(cellB);

      let comparison = 0;

      const numA = parseFloat(valueA);
      const numB = parseFloat(valueB);

      if (!isNaN(numA) && !isNaN(numB)) {
        comparison = numA - numB;
      } else {

        comparison = valueA.localeCompare(valueB, undefined, { numeric: true, sensitivity: 'base' });
      }

      return direction === 'asc' ? comparison : -comparison;
    });

    rows.forEach(row => tbody.appendChild(row));
  }

  function getCellValue(cell) {
    return cell.textContent.trim();
  }

  function restoreOriginalOrder(tbody, headerRow, originalOrder) {
    originalOrder.forEach(row => tbody.appendChild(row));
  }

  if (document.readyState === 'loading') {
    document.addEventListener('DOMContentLoaded', initSortableTables);
  } else {
    initSortableTables();
  }
})();