function drawTotalProfit(domID, source) {
    let datas = [];
    let labels = []
    let totalProfit = 0;
    let i = 0;
    for (d in source){
        totalProfit = source[d].TotalProfit;
        if (totalProfit === 0){
            continue;
        }
        datas.push(totalProfit);
        labels.push(source[d].Time);
        i++;
    }
    var ctx = document.getElementById(domID).getContext('2d');
    var myChart = new Chart(ctx, {
        type: 'line',
        data: {
            labels: labels,
            datasets: [{
                label: 'Total Profit',
                data: datas,
                backgroundColor: ['rgba(54, 162, 235, 0.2)',
                    'rgba(255, 206, 86, 0.2)',
                    'rgba(75, 192, 192, 0.2)',
                    'rgba(153, 102, 255, 0.2)',
                    'rgba(255, 159, 64, 0.2)']
            }],

        },
        options: {
        }
    });
}

function drawProfit(domID, source) {
    let datas = [];
    let labels = []
    let lastProfit = 0;
    let i = 0;
    for (d in source){
        let profit = source[d].Profit;
        if (profit == 0){
            continue;
        }
        datas.push(profit);
        lastProfit = profit;
        labels.push(source[d].Time);
        i++;
    }
    var ctx = document.getElementById(domID).getContext('2d');
    var myChart = new Chart(ctx, {
        type: 'line',
        data: {
            labels: labels,
            datasets: [{
                label: 'Profit',
                data: datas,
                backgroundColor: ['rgba(255, 99, 132, 0.2)',
                    'rgba(54, 162, 235, 0.2)',
                    'rgba(255, 206, 86, 0.2)',
                    'rgba(75, 192, 192, 0.2)',
                    'rgba(153, 102, 255, 0.2)',
                    'rgba(255, 159, 64, 0.2)']
            }],

        },
        options: {
        }
    });
}