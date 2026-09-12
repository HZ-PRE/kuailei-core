const {  sdmClient } = require('./client.js');
const sdm = require("./sdm_grpc_web_pb.js");

function openConnectionPage() {
    
        $("#extension-list-container").show();
        $("#extension-page-container").hide();
        $("#connection-page").show();
        connect();
        $("#connect-button").click(async () => {
            const hsetting_request = new sdm.ChangeSdmSettingsRequest();
            hsetting_request.setSdmSettingsJson($("#sdm-settings").val());
            try{
                const hres=await sdmClient.changeSdmSettings(hsetting_request, {});
            }catch(err){
                $("#sdm-settings").val("")
                console.log(err)
            }
            
            const parse_request = new sdm.ParseRequest();
            parse_request.setContent($("#config-content").val());
            try{
                const pres=await sdmClient.parse(parse_request, {});
                if (pres.getResponseCode() !== sdm.ResponseCode.OK){
                    alert(pres.getMessage());
                    return
                }
                $("#config-content").val(pres.getContent());
            }catch(err){
                console.log(err)
                alert(JSON.stringify(err))
                                return
            }

            const request = new sdm.StartRequest();
    
            request.setConfigContent($("#config-content").val());
            request.setEnableRawConfig(false);
            try{
                const res=await sdmClient.start(request, {});
                console.log(res.getCoreState(),res.getMessage())
                    handleCoreStatus(res.getCoreState());
            }catch(err){
                console.log(err)
                alert(JSON.stringify(err))
                return
            }

            
        })

        $("#disconnect-button").click(async () => {
            const request = new sdm.Empty();
            try{
                const res=await sdmClient.stop(request, {});
                console.log(res.getCoreState(),res.getMessage())
                handleCoreStatus(res.getCoreState());
            }catch(err){
                console.log(err)
                alert(JSON.stringify(err))
                return
            }
        })
}


function connect(){
    const request = new sdm.Empty();
    const stream = sdmClient.coreInfoListener(request, {});
    stream.on('data', (response) => {
        console.log('Receving ',response);
        handleCoreStatus(response);
    });
    
    stream.on('error', (err) => {
        console.error('Error opening extension page:', err);
        // openExtensionPage(extensionId);
    });
    
    stream.on('end', () => {
        console.log('Stream ended');
        setTimeout(connect, 1000);
        
    });
}


function handleCoreStatus(status){
    if (status == sdm.CoreState.STOPPED){
        $("#connection-before-connect").show();
        $("#connection-connecting").hide();
    }else{
        $("#connection-before-connect").hide();
        $("#connection-connecting").show();
        if (status == sdm.CoreState.STARTING){
            $("#connection-status").text("Starting");
            $("#connection-status").css("color", "yellow");
        }else if (status == sdm.CoreState.STOPPING){
            $("#connection-status").text("Stopping");
            $("#connection-status").css("color", "red");
        }else if (status == sdm.CoreState.STARTED){
            $("#connection-status").text("Connected");
            $("#connection-status").css("color", "green");
        }
    }
}


module.exports = { openConnectionPage };