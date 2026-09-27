(function(root){
 'use strict';
 function invalid(field,message){const error=new Error(message);error.field=field;throw error;}
 function deployment(source,options){
  const {broker,target,senseImage,labImage,instanceId='sense-edge-01'}=options;
  if(!/^[a-zA-Z0-9.-]+$/.test(broker||''))invalid('deployBroker','Enter the MQTT broker hostname or IPv4 address.');
  if(!/^[a-zA-Z0-9_][a-zA-Z0-9_.-]*@[a-zA-Z0-9][a-zA-Z0-9.-]*$/.test(target||''))invalid('deployTarget','Enter the VM login as user@IP, for example iot@192.168.1.30.');
  if(!/^[a-zA-Z0-9][a-zA-Z0-9_-]{0,63}$/.test(instanceId))invalid('deployInstance','Use letters, numbers, underscores or hyphens for the collector name.');
  for(const [field,image] of [['deploySenseImage',senseImage],['deployLabImage',labImage]])if(!/^[a-zA-Z0-9][a-zA-Z0-9._/:@-]*$/.test(image||''))invalid(field,'Enter a valid Docker image name.');
  if(!source?.type?.startsWith('lab-'))throw Error('Select a TCP connection first.');
  let endpoint;try{endpoint=new URL(source.options.connection.replace('modbus-rtu:tcp:','tcp:'));if(!endpoint.hostname)throw Error();}catch{invalid('deployConnection','Enter a complete device URL including protocol, host and port.');}
  if(['localhost','[::1]','0.0.0.0','protocol-lab'].includes(endpoint.hostname)||endpoint.hostname.startsWith('127.'))invalid('deployConnection','Device connection still points to the local demo address ('+endpoint.hostname+'). Enter the actual machine or gateway IP reachable from the VM.');
  const item=JSON.parse(JSON.stringify(source));item.options.gatewayURL='http://protocol-lab:8500';
  item.agentId=instanceId+'-source';
  const config={broker:{host:broker,port:1883,clientId:instanceId,topicPrefix:'iot-lense'},pollIntervalMs:1000,healthTimeoutSeconds:30,sources:[item]};
  const compose={services:{'protocol-lab':{image:labImage,restart:'unless-stopped'},sense:{image:senseImage,restart:'unless-stopped',ports:['8100:8100'],environment:{SENSE_CONFIG_PATH:'/app/config/config.json',OTIO_CONFIG_READ_ONLY:'true'},volumes:['./sense-config.json:/app/config/config.json:ro'],depends_on:['protocol-lab']}}};
  const helmSource=JSON.parse(JSON.stringify(item));helmSource.options.gatewayURL='http://sense-edge-gateway:8500';
  const values={senseImage,labImage,config:{...config,sources:[helmSource]}};
  const certificateProfile=endpoint.searchParams.get('certificateProfile');
  if(certificateProfile&&!/^[a-zA-Z0-9][a-zA-Z0-9_-]{0,63}$/.test(certificateProfile))invalid('deployConnection','Invalid certificate profile name.');
  const files={'sense-config.json':JSON.stringify(config,null,2),'compose.json':'','sense-values.json':''};
  if(certificateProfile){
   compose.services['protocol-lab'].environment={OTIO_CERTIFICATE_DIR:'/var/lib/otio/certificates'};
   compose.services['protocol-lab'].volumes=['./certificates:/var/lib/otio/certificates:ro'];
   values.existingCertificateSecret='sense-edge-certificates';
   files['prepare-certificate.py']=['# Put the ORIGINAL client.pem, client.key and server.pem beside this script.','# Private keys are never downloaded from Lense. Requires Python 3.','import json, os','from pathlib import Path',"os.umask(0o077)","Path('certificates').mkdir(exist_ok=True)","profile = {'id': '"+certificateProfile+"', 'certificate': Path('client.pem').read_text(), 'privateKey': Path('client.key').read_text(), 'serverCertificate': Path('server.pem').read_text()}","Path('certificates/"+certificateProfile+".json').write_text(json.dumps(profile))"].join('\n');
  }
  const commands=[
   ...(certificateProfile?['# Put client.pem, client.key and server.pem beside prepare-certificate.py.','python3 prepare-certificate.py']:[]),
   '# From the OT.io repository; build images matching the target CPU architecture:',
   'docker build -f apps/sense/Dockerfile -t '+senseImage+' .',
   'docker build -f apps/protocol-lab/Dockerfile -t '+labImage+' .',
   '# Download compose.json and sense-config.json into the current directory.',
   '# Optional local test: docker compose -f compose.json up -d',
   '# Transfer instead to the remote Docker host:',
   'docker save -o otio-images.tar '+senseImage+' '+labImage,
   'ssh '+target+' "mkdir -p otio-sense"',
   'scp otio-images.tar compose.json sense-config.json '+target+':otio-sense/',
   ...(certificateProfile?['scp -r certificates '+target+':otio-sense/']:[]),
   'ssh '+target+' "cd otio-sense && docker load -i otio-images.tar && docker compose -f compose.json up -d"'
  ].join('\n');
  files['compose.json']=JSON.stringify(compose,null,2);files['sense-values.json']=JSON.stringify(values,null,2);
  return {files,commands,helm:(certificateProfile?'# Run python3 prepare-certificate.py first.\nkubectl create secret generic sense-edge-certificates --from-file='+certificateProfile+'.json=certificates/'+certificateProfile+'.json --dry-run=client -o yaml | kubectl apply -f -\n':'')+'# From the OT.io repository. Push both images to your registry first,\n# or load both into every local cluster node. Download sense-values.json.\nhelm upgrade --install sense-edge deploy/helm/otio-sense -f sense-values.json\nkubectl port-forward service/sense-edge 8100:8100'};
 }
 const api={deployment};root.IoTDeployment=api;if(typeof module==='object'&&module.exports)module.exports=api;
})(typeof window==='undefined'?globalThis:window);
