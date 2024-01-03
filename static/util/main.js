var halfHourSelected = false;

document.addEventListener('DOMContentLoaded', function() {
    document.getElementById('halfHourBtn').addEventListener('click', toggleHalfHour);
    document.getElementById('bookingForm').addEventListener('submit', submitForm);
});

function submitForm(event) {
    event.preventDefault();

    var form = document.getElementById('bookingForm');
    var hiddenInput = form.querySelector('input[name="halfHourSelected"]');

    if (!hiddenInput) {
        hiddenInput = document.createElement('input');
        hiddenInput.type = 'hidden';
        hiddenInput.name = 'halfHourSelected';
        form.appendChild(hiddenInput);
    }

    hiddenInput.value = halfHourSelected ? 'true' : 'false';

    console.log('Half hour selected:', halfHourSelected);

    form.submit();
}

function toggleHalfHour() {
    halfHourSelected = !halfHourSelected;
    
    var btn = document.getElementById('halfHourBtn');
    console.log('Half hour selected after toggle:', halfHourSelected);

    if (halfHourSelected) {
        btn.style.backgroundColor = '#5948D6';
        btn.value = 'Remove Half Hour';
    } else {
        btn.style.backgroundColor = '#6C63FF';
        btn.value = 'Add a Half Hour';
    }
}
