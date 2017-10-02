# Misc Notes

## Building the Docker Images
backend:
./build.sh
docker build -t dvonthenen/jop-backend .
docker push dvonthenen/jop-backend:latest

docker build -t dvonthenen/jop-backend-zipkin .
docker push dvonthenen/jop-backend-zipkin:latest

frontend:
./build.sh
docker build -t dvonthenen/jop-frontend .
docker push dvonthenen/jop-frontend:latest

docker build -t dvonthenen/jop-frontend-zipkin .
docker push dvonthenen/jop-frontend-zipkin:latest


## ScaleIO Clean Up
scli --login --username admin --password Scaleio123
scli --query_all_volumes
scli --unmap_volume_from_sdc --all_sdc --volume_name XXXXXX --i_am_sure
scli --remove_volume --volume_name XXXX

scli --mdm_ip 10.138.0.7 --add_volume --size_gb 16 --volume_name prometheus --protection_domain_name default --storage_pool_name default

scli --mdm_ip 10.138.0.7 --add_volume --size_gb 8 --volume_name statichttpserver1 --protection_domain_name default --storage_pool_name default

scli --mdm_ip 10.138.0.7 --add_volume --size_gb 8 --volume_name statichttpserver2 --protection_domain_name default --storage_pool_name default

scli --mdm_ip 10.138.0.7 --add_volume --size_gb 8 --volume_name statichttpserver3 --protection_domain_name default --storage_pool_name default

scli --mdm_ip 10.138.0.7 --add_volume --size_gb 8 --volume_name statichttpserver4 --protection_domain_name default --storage_pool_name default

scli --mdm_ip 10.138.0.7 --add_volume --size_gb 8 --volume_name statichttpserver5 --protection_domain_name default --storage_pool_name default

scli --mdm_ip 10.138.0.7 --add_volume --size_gb 8 --volume_name statichttpserver6 --protection_domain_name default --storage_pool_name default

scli --remove_volume --volume_name statichttpserver5


## Random Debugging Stuff
docker ps --filter "status=exited" | awk '{print $1}' | xargs docker rm

docker run -ti --entrypoint=/bin/sh -p 9090:9090 prom/prometheus:v1.7.1

/bin/prometheus -config.file=/etc/prometheus/prometheus.yml -storage.local.path=/prometheus -web.console.libraries=/etc/prometheus/console_libraries -web.console.templates=/etc/prometheus/consoles

IMPORTANT: Dont forget to fix ports on GCE!!!!!
