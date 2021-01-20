function main() {
    // 状态
    var status = []

    function mainLoop() {
        GetStatus()
        getstatus = setInterval(GetStatus, 5000)
    }

    function GetStatus() {
        $.ajax({
            url: "../status",
            
            success: function (resp) {
                // console.log(resp);
                if (resp == null) return;
                resp.sort(function (a, b) {
                    return a.name.localeCompare(b.name)
                })
                status = resp
                $('#target-list').html('')
                $('#target-list')
                .append(
                    $("<tr></tr>")
                    .append($('<th class="target-header"></th>').text('主机'))
                    .append($('<th class="target-header"></th>').text('地址'))
                    .append($('<th class="target-header"></th>').text('地区'))
                    .append($('<th class="target-header"></th>').text('省份'))
                    .append($('<th class="target-header"></th>').text('上次启动'))
                    .append($('<th class="target-header"></th>').text('在线时间'))
                    .append($('<th class="target-header"></th>').text('内存'))
                    .append($('<th class="target-header"></th>').text('磁盘'))
                    .append($('<th class="target-header"></th>').text('上行'))
                    .append($('<th class="target-header"></th>').text('下行'))
                    .append($('<th class="target-header"></th>').text('读'))
                    .append($('<th class="target-header"></th>').text('写'))
                    .append($('<th class="target-header"></th>').text('cpu')))
                for (var ti of resp) {
                    let lastStatus = ti.list[ti.list.length-1];
                    let lastStatusPre = ti.list[ti.list.length-2];

                    $('#target-list')
                        .append(
                            $('<tr class="target-item-row"></tr>')
                            .append($('<td class="target-col-item"></td>').text(ti.name))
                            .append($('<td class="target-col-item"></td>').text(lastStatus.Ip))
                            .append($('<td class="target-col-item"></td>').text(lastStatus.IpCountry))
                            .append($('<td class="target-col-item"></td>').text(lastStatus.IpRegion))
                            .append($('<td class="target-col-item"></td>').text(new Date(lastStatus.BootTime * 1000).Format("yyyy-MM-dd hh:mm:ss")))
                            .append($('<td class="target-col-item"></td>').text((lastStatus.Uptime / 60 / 60).toFixed(2) + "小时"))
                            .append($('<td class="target-col-item"></td>')
                                .append($('<div class="progress"></div>')
                                    .append($('<div style="width:'+lastStatus.UsedPercentMem.toFixed(2)+'%;"></div>')
                                        .append($('<span></span>').text((lastStatus.UsedMem / 1024 / 1024 / 1024).toFixed(2) + "Gb/" + (lastStatus.TotalMem / 1024 / 1024 / 1024).toFixed(2) + "Gb"))
                                    )
                                )
                            )
                            .append($('<td class="target-col-item"></td>')
                                .append($('<div class="progress"></div>')
                                    .append($('<div style="width:'+lastStatus.UsedPercentDisk.toFixed(2)+'%;"></div>')
                                        .append($('<span></span>').text((lastStatus.UsedDisk / 1024 / 1024 / 1024).toFixed(2) + "Gb/" + (lastStatus.TotalDisk / 1024 / 1024 / 1024).toFixed(2) + "Gb"))
                                    )
                                )
                            )
                            .append($('<td class="target-col-item"></td>').text(((lastStatus.NetStatus.map(a => a.ByteSend == undefined ? 0 : a.ByteSend ).reduce((a, b) => a + b) - lastStatusPre.NetStatus.map( a => a.ByteSend == undefined ? 0 : a.ByteSend).reduce((a, b) => a + b)) / (lastStatus.CheckTime - lastStatusPre.CheckTime) / 1024).toFixed(2) + " kb/s"))
                            .append($('<td class="target-col-item"></td>').text(((lastStatus.NetStatus.map(a => a.BytesRecv == undefined ? 0 : a.BytesRecv).reduce((a, b) => a + b) - lastStatusPre.NetStatus.map(a => a.BytesRecv == undefined ? 0 : a.BytesRecv).reduce((a, b) => a + b)) / (lastStatus.CheckTime - lastStatusPre.CheckTime) / 1024).toFixed(2) + " kb/s"))
                            .append($('<td class="target-col-item"></td>').text(((lastStatus.DiskStatus.map(a => a.ReadBytes == undefined ? 0 : a.ReadBytes).reduce((a, b) => a + b) - lastStatusPre.DiskStatus.map(a =>   a.ReadBytes == undefined ? 0 : a.ReadBytes).reduce((a, b) => a + b)) / (lastStatus.CheckTime - lastStatusPre.CheckTime) / 1024).toFixed(2) + " kb/s"))
                            .append($('<td class="target-col-item"></td>').text(((lastStatus.DiskStatus.map(a => a.WriteBytes == undefined ? 0 : a.WriteBytes).reduce((a, b) => a + b) - lastStatusPre.DiskStatus.map(a => a.WriteBytes == undefined ? 0 : a.WriteBytes).reduce((a, b) => a + b)) / (lastStatus.CheckTime - lastStatusPre.CheckTime) / 1024).toFixed(2) + " kb/s"))
                            .append($('<td class="target-col-item"></td>')
                                .append(lastStatus.CPUpr.map(a => $('<div class="cpu-progress"></div>')
                                    .append($('<div class="progress-cpu"></div>')
                                        .append($('<div style="width:' + a/100 + '%;"></div>')
                                            .append($('<span></span>').text(a.toFixed(2) + "%"))
                                        )
                                    )
                                )
                            )
                        )
                    )
                }
            }
        })
    }

    mainLoop()
}

main()



