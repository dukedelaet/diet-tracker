const tooltipData = [{{.Tooltips}}];
new Chart(document.getElementById('chart'), {
    type: 'line',
    data: {
        labels: [{{.Labels}}],
        datasets: [{
            label: 'Exercise',
            data: [{{.Data}}],
            borderColor: 'rgb(153, 102, 255)',
            backgroundColor: 'rgba(153, 102, 255, 0.3)',
            pointBackgroundColor: 'rgb(153, 102, 255)',
            pointRadius: 8,
            pointHoverRadius: 12,
            tension: 0,
            fill: true,
            stepped: true
        }]
    },
    options: {
        responsive: true,
        plugins: {
            legend: { display: false },
            tooltip: {
                callbacks: {
                    label: function(context) {
                        return tooltipData[context.dataIndex];
                    }
                }
            }
        },
        scales: {
            y: {
                min: 0,
                max: 1.5,
                ticks: {
                    callback: function(value) {
                        return value === 1 ? 'Exercised' : '';
                    }
                },
                title: { display: true, text: 'Activity' }
            },
            x: {
                title: { display: true, text: 'Date' }
            }
        }
    }
});
