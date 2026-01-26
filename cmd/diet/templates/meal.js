new Chart(document.getElementById('chart'), {
    type: 'line',
    data: {
        labels: [{{.Labels}}],
        datasets: [{
            label: 'Calories',
            data: [{{.Data}}],
            borderColor: 'rgb(255, 159, 64)',
            backgroundColor: 'rgba(255, 159, 64, 0.1)',
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
                beginAtZero: true,
                title: { display: true, text: 'Calories' }
            },
            x: {
                title: { display: true, text: 'Date' }
            }
        }
    }
});
