new Chart(document.getElementById('chart'), {
    type: 'line',
    data: {
        labels: [{{.Labels}}],
        datasets: [{
            label: 'Weight (lbs)',
            data: [{{.Data}}],
            borderColor: 'rgb(75, 192, 192)',
            backgroundColor: 'rgba(75, 192, 192, 0.1)',
            tension: 0.1,
            fill: true
        }]
    },
    options: {
        responsive: true,
        plugins: {
            legend: { display: true, position: 'top' }
        },
        scales: {
            y: {
                beginAtZero: false,
                title: { display: true, text: 'Pounds' }
            },
            x: {
                title: { display: true, text: 'Date' }
            }
        }
    }
});
