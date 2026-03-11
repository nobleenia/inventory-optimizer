// Inventory Optimizer — UI interactions.
document.addEventListener('DOMContentLoaded', function () {

    // Show chosen file names.
    var salesInput = document.getElementById('sales_file');
    var paramsInput = document.getElementById('params_file');

    if (salesInput) {
        salesInput.addEventListener('change', function () {
            var name = this.files[0] ? this.files[0].name : 'No file chosen';
            document.getElementById('sales-name').textContent = name;
            this.closest('.file-group').classList.toggle('has-file', !!this.files[0]);
        });
    }

    if (paramsInput) {
        paramsInput.addEventListener('change', function () {
            var name = this.files[0] ? this.files[0].name : 'No file chosen';
            document.getElementById('params-name').textContent = name;
            this.closest('.file-group').classList.toggle('has-file', !!this.files[0]);
        });
    }

    // Loading state on form submit.
    var form = document.getElementById('upload-form');
    if (form) {
        form.addEventListener('submit', function () {
            var btn = document.getElementById('submit-btn');
            if (btn) {
                btn.disabled = true;
                btn.querySelector('.btn-text').style.display = 'none';
                btn.querySelector('.btn-loading').style.display = 'inline';
            }
        });
    }
});
